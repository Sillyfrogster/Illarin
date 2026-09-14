package extension

// approvedByAdmin holds the permissions Lumiverse will not grant until an admin approves them.
var approvedByAdmin = map[string]bool{
	"app_manipulation": true, "cors_proxy": true, "generation": true, "interceptor": true,
	"context_handler": true, "macro_interceptor": true, "characters": true, "chats": true,
	"world_books": true, "presets": true, "regex_scripts": true,
	"regex_scripts_unrestricted": true, "databanks": true, "personas": true,
	"push_notification": true, "image_gen": true, "images": true, "web_search": true,
	"unsafe_eval": true, "providers.embedding.register": true, "providers.tts.register": true,
	"providers.stt.register": true, "providers.sidecar.register": true, "mcp_servers": true,
	"mcp_servers.create": true,
}

var permissionWords = map[string]string{
	"app_manipulation":           "Place its own interface anywhere in the app and change the theme.",
	"cors_proxy":                 "Send web requests through your Lumiverse server.",
	"generation":                 "Run text generations for you and see your connection profiles.",
	"interceptor":                "Change a prompt before it reaches the model.",
	"context_handler":            "Add to what the model is given before the prompt is built.",
	"characters":                 "Read, create, change and delete your characters.",
	"chats":                      "Read, change and delete your chats.",
	"world_books":                "Read, create, change and delete your world books.",
	"presets":                    "Read, create, change and delete your presets.",
	"regex_scripts":              "Read your regex scripts and manage the ones it made.",
	"regex_scripts_unrestricted": "Change or delete any of your regex scripts.",
	"databanks":                  "Read, create, change and delete your databanks.",
	"personas":                   "Read, create, change, delete and switch your personas.",
	"push_notification":          "Send notifications to your devices while the app is closed.",
	"image_gen":                  "Generate images with your image connections.",
	"images":                     "Read, upload and delete your stored images and videos.",
	"web_search":                 "Search the web through your search provider.",
	"tools":                      "Give the model tools it can call.",
	"generation_parameters":      "Change the settings sent with a generation.",
	"ephemeral_storage":          "Keep temporary data that expires.",
	"memories":                   "Read and change the Memory Cortex and long-term chat memory.",
	"chat_mutation":              "Read, add, change and hide chat messages.",
	"event_tracking":             "Record and replay its own usage events.",
	"ui_panels":                  "Open floating widgets and docked panels.",
	"oauth":                      "Receive sign-in redirects from other services.",
	"media":                      "Convert and combine audio and video.",
}

const undescribedPermission = "Illarin has no description of this permission."

func describePermission(name string) string {
	if words, known := permissionWords[name]; known {
		return words
	}
	return undescribedPermission
}
