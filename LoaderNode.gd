extends Node

var http_request_node
var http_downloader_node
var resource_pack_file
var main_scene_file

func _ready():
	# Maybe not have a static name here? Plus is this relative "path" okay?
	resource_pack_file = 'nextgame.pck'
	
	http_request_node = HTTPRequest.new()
	http_request_node.connect("request_completed", self, "_http_request_next_completed")
	add_child(http_request_node)
	
	http_downloader_node = HTTPRequest.new()
	http_downloader_node.connect("request_completed", self, "_http_request_download_completed")
	add_child(http_downloader_node)

# Call this from wherever
func load_next():
	var platform = "windowsdesktop"
	if OS.get_name() == "X11":
		platform = "linuxx11"
		
	var game_name = ProjectSettings["application/config/name"]
		
	var error = http_request_node.request("https://devnullga.me/godot-delivery/next-game?platform=" + platform + "&gamename=" + game_name)
	if error != OK:
		push_error("An error occurred in the HTTP request to load the next level identifier.")

func _http_request_next_completed(result, response_code, headers, body):
	if response_code >= 400:
		push_error("Next level loader got BadRequest")
		return
	
	var response = parse_json(body.get_string_from_utf8())
	var download_url = response["download_url"]
	main_scene_file = response["main_scene"]
	
	http_downloader_node.download_file = resource_pack_file
	http_downloader_node.request(download_url)

func _http_request_download_completed(result, response_code, headers, body):
	if response_code >= 400:
		push_error("Next level downloader got BadRequest")
		return
		
	var success = ProjectSettings.load_resource_pack(resource_pack_file)
	if !success:
		push_error("Unable to load resource pack")
		return
		
	var error = get_tree().change_scene(main_scene_file)
	if error != OK:
		push_error("Unable to change to new scene")
