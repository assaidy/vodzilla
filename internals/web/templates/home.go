package templates

import . "github.com/assaidy/hyper/v2"

func HomePage() HyperNode {
	return pageWithSse("Home", Group(
		H1(AttrClass("text-5xl"))("Hello, World"),
		BUTTON(AttrClass("btn"), Attr("data-on:click", "@get('/test')"))("test"),
	))
}
