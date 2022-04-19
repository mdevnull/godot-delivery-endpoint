# Dynamic Level Loader

This project contains the server side code for a delivery endpoint that creates PCK files from git repositories that contain a godot project as well as provides an REST like API to query and download PCKs stored on the server.

## Server Setup

This repository also provides a docker image. You may use that image to setup the server. Below find a list of environment variables to configure the server.

|Env Name|Description|Example|
|--|--|--|
|STORAGE_PATH|Absolute path to storage directory|/path/to/storage|
|BASE_URL|Base URL of the external HTTP server|http://localhost:8082/
|AUTH_PW|Password for add-repository endpoint. Username is fixed to "manager"|changeme|
|WEBADDRESS|Port Binding for webserver|:8082|

## Add Games to the rotation

After the server setup is complete you can add git repositories by calling `godot-delivery/add-repository` with a body payload like this:

```json
{
    "repository": "https://github.com/devnull-twitch/level-loading-tests.git"
}
```

The server will checkout the repository and build all available export presets.

__Note: The repository has to be public.__ 

## Updating game PCKs

If the source of the game has changed ( new commits in repository ) the server will __not__ update automatically.
You need to make the call to `godot-delivery/add-repository` again. With the same repository URL. That will overwrite the PCKs.

## Client Setup

You can use the Node GD script implementation in `LoaderNode.gd`.
That script uses the internal Godot HTTPRequest node. So its slow. Like really slow.