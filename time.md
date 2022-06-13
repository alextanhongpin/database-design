# Representing date time logic in postgres


1) marking date of operations. Timestamp fields like created at, updated at. Also, instead of using bool, we can always use timestamp to represent changes, such as a products published at,or a token confirmed at.
2) for time periods, use tstzrange. This is suitable for stuff that is valid for a specific time, or things that needs tobe scheduled. However, avoid using this if dealing with continuous data
3) continuos data represents data that will always be valid, but might change over time
