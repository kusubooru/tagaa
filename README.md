# local-tagger
User interface for the 'Bulk Add CSV' extension of Shimmie2.

local-tagger will launch a web interface in a new browser window, which
allows to add tags, source and rating on each image that is contained in the
current directory (or the one specified by the -dir option). Subfolders are
ignored. Supported types: "gif", "jpeg", "jpg", "png", "swf"

The web interface allows to save the image metadata in a CSV file as expected
by the 'Bulk Add CSV' Shimmie2 extension. If a CSV file with the name
'bulk.csv' (or a name specified by the -csv option) is found, it will be
loaded automatically on start up.

The folder containing the CSV file and the images can then be manually
uploaded to the server and used by the 'Bulk Add CSV' extension to bulk add
the images to Shimmie2.

