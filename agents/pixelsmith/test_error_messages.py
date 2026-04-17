from pixelsmith.pixelsmith import friendly_generation_error


def test_center_background_cleanup_maps_to_friendly():
    msg = friendly_generation_error("item sprite failed deterministic sprite gates: center_background_cleanup")
    assert "background" in msg.lower() or "busy" in msg.lower() or "simpler" in msg.lower(), msg


def test_min_contrast_maps_to_friendly():
    msg = friendly_generation_error("item sprite failed deterministic sprite gates: min_contrast_check")
    assert "contrast" in msg.lower() or "color" in msg.lower(), msg


def test_aspect_ratio_maps_to_friendly():
    msg = friendly_generation_error("item sprite failed deterministic sprite gates: aspect_ratio_check")
    assert "size" in msg.lower() or "ratio" in msg.lower() or "shape" in msg.lower(), msg


def test_unknown_error_passes_through():
    msg = friendly_generation_error("some unexpected internal error")
    assert "unexpected" in msg.lower() or "some" in msg.lower(), msg


def test_fal_key_missing_maps_to_friendly():
    msg = friendly_generation_error("FAL_KEY (or FAL_API_KEY) is required for Pixelsmith.")
    assert "api key" in msg.lower() or "fal" in msg.lower(), msg
