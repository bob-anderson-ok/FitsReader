# FitsReader   - an Iota utility

This program is both a general purpose "viewer" and a utility to process
    a "flash timed" recording produced by the SharCap/IotaGFTapp duo for the
    purpose of inserting GPS accurate timestamps in each fits file.

    If enable-auto-timestamp-insertion is checked, opening a fits folder produced by
    the SharpCap/IotaGFTapp duo will be automatically timestamp-insertion-processed.

    Timestamp insertion will only occur once.

    The vertical sliders at the right control black and white image levels for
    contrast enhancement. The program applies an initial setting pair by analyzing
    the statistics of the first image.

    The left slider sets minimum black level. The right slider sets maximum white level.

    If you set the min black above the max white, the image will be inverted - some
    people may find this easier to look at.

    A ROI (region of interest) can be used to "zoom" in on an area of special interest
    in the image.

    Looping can be done backwards as well as (the usual) forwards.

    Playback should be paused when setting ROI dimensions.

    Author: Bob Anderson  bob.anderson.ok@gmail.com (also PyMovie, PyOTE, R-ote, Occular)

    1 June 2024


