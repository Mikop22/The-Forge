import asyncio
import json
import unittest
from pathlib import Path
from tempfile import TemporaryDirectory
from unittest import mock

from watchdog.events import FileCreatedEvent, FileMovedEvent

import orchestrator


class RequestHandlerEventTests(unittest.TestCase):
    def setUp(self) -> None:
        self.loop = asyncio.new_event_loop()
        self.handler = orchestrator._RequestHandler(self.loop)

    def tearDown(self) -> None:
        self.loop.close()

    def test_created_request_file_triggers_pipeline(self) -> None:
        with TemporaryDirectory() as tmpdir:
            request_file = Path(tmpdir) / "user_request.json"
            request_file.write_text(json.dumps({"prompt": "Blade"}), encoding="utf-8")

            with mock.patch.object(orchestrator, "REQUEST_FILE", request_file), \
                 mock.patch.object(orchestrator.asyncio, "run_coroutine_threadsafe") as runner:
                self.handler._last_trigger = 0.0
                self.handler.on_created(FileCreatedEvent(str(request_file)))

            self.assertEqual(runner.call_count, 1)
            runner.call_args.args[0].close()

    def test_moved_request_file_triggers_pipeline(self) -> None:
        with TemporaryDirectory() as tmpdir:
            request_file = Path(tmpdir) / "user_request.json"
            request_file.write_text(json.dumps({"prompt": "Blade"}), encoding="utf-8")
            tmp_request = request_file.with_suffix(".tmp")
            tmp_request.write_text(json.dumps({"prompt": "Blade"}), encoding="utf-8")

            with mock.patch.object(orchestrator, "REQUEST_FILE", request_file), \
                 mock.patch.object(orchestrator.asyncio, "run_coroutine_threadsafe") as runner:
                self.handler._last_trigger = 0.0
                self.handler.on_moved(FileMovedEvent(str(tmp_request), str(request_file)))

            self.assertEqual(runner.call_count, 1)
            runner.call_args.args[0].close()


if __name__ == "__main__":
    unittest.main()
