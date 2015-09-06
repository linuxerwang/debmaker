deb-spec {
	control {
		pkg-name: "myapp"
		maintainer: "linuxerwang@gmail.com"
		description: "My application."

		other-attrs: {
			"Section": "utils",
			"Priority": "optional",
		}
	}

	debian {
		path: "example/preinst"
		deb-path: "preinst"
	}

	content {
		path: "example/README.txt"
		deb-path: "usr/share/myapp/README"
	}

	content {
		path: "example/scripts"
		deb-path: "usr/share/myapp/scripts"
	}

	link {
		from: "/usr/share/myapp/scripts/processor.js"
		to: "/usr/share/myapp/scripts/processor-firefox.js"
	}
}
