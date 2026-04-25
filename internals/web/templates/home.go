package templates

import . "github.com/assaidy/hyper/v2"

func HomePage() HyperNode {
	return page("Home", Group(
		H1(AttrClass("text-5xl"))("Hello, World"),
	))
}
