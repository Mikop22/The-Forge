from pathlib import Path


def test_connector_writes_runtime_summary_file() -> None:
    source = Path("mod/ForgeConnector/ForgeConnectorSystem.cs").read_text(encoding="utf-8")
    assert "forge_runtime_summary.json" in source
    assert "WriteRuntimeSummary(" in source
    assert "live_item_name" in source
    assert "last_runtime_note" in source


def test_connector_does_not_mark_failed_item_as_live() -> None:
    source = Path("mod/ForgeConnector/ForgeConnectorSystem.cs").read_text(encoding="utf-8")
    assert 'UpdateRuntimeSummaryState("inject_failed", runtimeNote:' in source
    assert 'UpdateRuntimeSummaryState("inject_failed", itemName' not in source


def test_runtime_summary_reverts_to_menu_note_when_not_in_world() -> None:
    source = Path("mod/ForgeConnector/ForgeConnectorSystem.cs").read_text(encoding="utf-8")
    assert "string note = worldLoaded" in source
    assert ': "At main menu."' in source
