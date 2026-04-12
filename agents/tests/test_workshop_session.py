from core.workshop_session import WorkshopSessionStore


def test_session_store_round_trips_bench_and_shelf(tmp_path) -> None:
    store = WorkshopSessionStore(tmp_path)
    store.save(
        {
            "session_id": "sess-1",
            "bench": {"item_id": "storm-brand", "label": "Storm Brand"},
            "shelf": [{"variant_id": "v1", "label": "Heavier Shot"}],
        }
    )

    loaded = store.load("sess-1")
    assert loaded["bench"]["item_id"] == "storm-brand"
    assert loaded["shelf"][0]["variant_id"] == "v1"


def test_session_store_tracks_active_session(tmp_path) -> None:
    store = WorkshopSessionStore(tmp_path)
    store.save({"session_id": "sess-1", "bench": {"item_id": "storm-brand"}})
    store.save({"session_id": "sess-2", "bench": {"item_id": "orbit-furnace"}})

    assert store.active_session_id() == "sess-2"
    assert store.load_active()["bench"]["item_id"] == "orbit-furnace"
