using System;
using System.IO;
using System.Reflection;
using System.Text.Json;
using System.Threading;
using Terraria;
using Terraria.ModLoader;

namespace ForgeConnector
{
    /// <summary>
    /// ModSystem that watches for command_trigger.json and triggers a mod reload
    /// on the main thread via PostUpdateEverything().
    /// </summary>
    public class ForgeConnectorSystem : ModSystem
    {
        private static int _reloadRequested = 0; // 1 = reload pending; Interlocked for thread safety

        private FileSystemWatcher? _watcher;
        private string _modSourcesDir = string.Empty;
        private string _triggerPath = string.Empty;
        private string _heartbeatPath = string.Empty;

        // ------------------------------------------------------------------
        // Lifecycle
        // ------------------------------------------------------------------

        public override void PostSetupContent()
        {
            _modSourcesDir = GetModSourcesDir();
            _triggerPath   = Path.Combine(_modSourcesDir, "command_trigger.json");
            _heartbeatPath = Path.Combine(_modSourcesDir, "forge_connector_alive.json");

            WriteHeartbeat();
            StartWatcher();
        }

        public override void Unload()
        {
            _watcher?.Dispose();
            _watcher = null;

            try { File.Delete(_heartbeatPath); } catch { /* best-effort */ }
        }

        // ------------------------------------------------------------------
        // Main-thread hook
        // ------------------------------------------------------------------

        public override void PostUpdateEverything()
        {
            if (Interlocked.Exchange(ref _reloadRequested, 0) == 1)
            {
                TriggerReload();
            }
        }

        // ------------------------------------------------------------------
        // FileSystemWatcher (runs on threadpool — no game API calls here)
        // ------------------------------------------------------------------

        private void StartWatcher()
        {
            if (!Directory.Exists(_modSourcesDir))
                return;

            _watcher = new FileSystemWatcher(_modSourcesDir, "command_trigger.json")
            {
                NotifyFilter = NotifyFilters.FileName | NotifyFilters.LastWrite,
                EnableRaisingEvents = true,
            };

            _watcher.Created += OnTriggerEvent;
            _watcher.Changed += OnTriggerEvent;
        }

        private void OnTriggerEvent(object sender, FileSystemEventArgs e)
        {
            // macOS APFS can fire double events — the delete-after-read below prevents acting twice.
            Thread.Sleep(50); // let the writer finish flushing

            try
            {
                string json = File.ReadAllText(_triggerPath);
                using JsonDocument doc = JsonDocument.Parse(json);
                if (!doc.RootElement.TryGetProperty("action", out JsonElement actionEl))
                    return;

                if (actionEl.GetString() != "execute")
                    return;

                // Delete immediately to prevent double-trigger on APFS double-fire.
                try { File.Delete(_triggerPath); } catch { /* best-effort */ }

                Interlocked.Exchange(ref _reloadRequested, 1);
            }
            catch
            {
                // Swallow — file may still be mid-write; next event will retry.
            }
        }

        // ------------------------------------------------------------------
        // Reload trigger (called from main thread only)
        // ------------------------------------------------------------------

        private static void TriggerReload()
        {
            // Strategy 1: Public menu-path reload (preferred, no reflection).
            // The reloadModsID field lives in different namespaces across tML versions;
            // try both common locations.
            try
            {
                // tModLoader 1.4.4+
                var interfaceType = Type.GetType("Terraria.ModLoader.UI.Interface, tModLoader");
                if (interfaceType == null)
                    interfaceType = Type.GetType("Terraria.GameContent.UI.Interface, Terraria");

                if (interfaceType != null)
                {
                    var field = interfaceType.GetField("reloadModsID",
                        BindingFlags.Static | BindingFlags.Public | BindingFlags.NonPublic);

                    if (field != null)
                    {
                        int reloadId = (int)field.GetValue(null)!;
                        Main.menuMode = reloadId;
                        return;
                    }
                }
            }
            catch { /* fall through to strategy 2 */ }

            // Strategy 2: Reflection into ModLoader.Reload() (non-public static).
            try
            {
                typeof(ModLoader)
                    .GetMethod("Reload", BindingFlags.Static | BindingFlags.NonPublic)
                    ?.Invoke(null, null);
            }
            catch { /* nothing we can do */ }
        }

        // ------------------------------------------------------------------
        // Helpers
        // ------------------------------------------------------------------

        private static string GetModSourcesDir()
        {
            string home = Environment.GetFolderPath(Environment.SpecialFolder.UserProfile);
            return Path.Combine(home, "Documents", "My Games", "Terraria", "tModLoader", "ModSources");
        }

        private void WriteHeartbeat()
        {
            try
            {
                Directory.CreateDirectory(_modSourcesDir);

                var payload = new
                {
                    status     = "listening",
                    loaded_at  = DateTime.UtcNow.ToString("o"),
                    pid        = Environment.ProcessId,
                };

                string json = JsonSerializer.Serialize(payload, new JsonSerializerOptions { WriteIndented = true });

                // Atomic write: write to .tmp then rename.
                string tmp = _heartbeatPath + ".tmp";
                File.WriteAllText(tmp, json);
                File.Move(tmp, _heartbeatPath, overwrite: true);
            }
            catch { /* best-effort */ }
        }
    }
}
