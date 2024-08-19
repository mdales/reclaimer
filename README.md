
# Reclaimer: download source data

A command-line tool for downloading scientific data from source repositories. Currently supported are:

* Zenodo
* Copernicus Land Monitoring Service (CLMS)


## Zenodo

Zenodo is a common place for results of papers to be published. Here you can download assests using the Zenodo ID, optionally specifying which files from the archive you want.

## Copernicus Land Monitoring Service

The CLMS API is stateful, notably you generally need to request data, then wait for the CLMS servers to make it available later. So the tool can both poll the server for updates and resume previously started requests. It also supports direct download for raw datasets when supported.
