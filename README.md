# Pollen

Reads pollen data from opendata.dwd.de and shows a simple web page with the data.

There is a Dockerfile to create the smallest possible docker image and a docker-compose.yaml to setup the complete app.

## build

Clone this repository go to the cloned sources and run:

    go get ./...
    go build .

This will build the pollen executable.

In templates are the Golang html **templates** for the different pages. In **assets** is the static content like style sheets and images. 

## run

The executable needs no parameter. It reads the data from http://opendata.dwd.de/climate_environment/health/alerts/s31fg.json and serves the web pages on port 8080. So front page can be reached with http://localhost:8080.
The following environment variables can be defined:

POLLEN_PORT
The ip port on which pollen will listen. Default is `8080`.

POLLEN_TEMPLATES_DIR
The directory where pollen looks for the template files. Default is `templates`.

POLLEN_ASSETS_DIR
The directory where pollen looks for the asset files. Default is `assets`.

## Docker

The Dockerfile works in two steps. The first step will build the executable, the second step creates the image from scratch.
The docker-compose.yaml can build the poller image and creates the container.
