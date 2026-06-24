package templates

import . "github.com/assaidy/hyper/v2"

var (
	// AttrHxGet issues an HTTP GET request to the specified URL when the element is triggered.
	AttrHxGet = MakePairAttribute("hx-get")
	// AttrHxPost issues an HTTP POST request to the specified URL when the element is triggered.
	AttrHxPost = MakePairAttribute("hx-post")
	// AttrHxPut issues an HTTP PUT request to the specified URL when the element is triggered.
	AttrHxPut = MakePairAttribute("hx-put")
	// AttrHxPatch issues an HTTP PATCH request to the specified URL when the element is triggered.
	AttrHxPatch = MakePairAttribute("hx-patch")
	// AttrHxDelete issues an HTTP DELETE request to the specified URL when the element is triggered.
	AttrHxDelete = MakePairAttribute("hx-delete")
	// AttrHxTarget specifies the target DOM element to be updated by the AJAX response.
	// Accepts CSS selectors (e.g., "#my-div", ".my-class") or htmx keywords (e.g., "this", "closest form").
	AttrHxTarget = MakePairAttribute("hx-target")
	// AttrHxDisable completely disables htmx processing for the element and all of its children.
	AttrHxDisable = MakePairAttribute("hx-disable")
	// AttrHxIndicator specifies a DOM element (usually a loading spinner) to show/hide
	// while the AJAX request is in flight.
	AttrHxIndicator = MakePairAttribute("hx-indicator")
	// AttrHxSwap controls how the AJAX response HTML is injected into the DOM.
	// Common options in htmx 4 include: "innerHTML", "outerHTML", "innerMorph", "outerMorph", or "none".
	AttrHxSwap = MakePairAttribute("hx-swap")
	// AttrHxSwapOob enables Out-of-Band swaps. It allows response HTML to swap directly into
	// specific targets elsewhere on the page, bypassing the main hx-target.
	AttrHxSwapOob = MakePairAttribute("hx-swap-oob")
	// AttrHxConfirm prompts the user with a browser confirmation dialog before issuing a request.
	// If the user cancels, the AJAX request is aborted.
	AttrHxConfirm = MakePairAttribute("hx-confirm")
	// AttrHxTrigger specifies the event or condition that triggers the AJAX request.
	// Examples: "load", "click consume", "intersect", "change", "keyup delay:200ms".
	AttrHxTrigger = MakePairAttribute("hx-trigger")
	// AttrHxPushUrl pushes a new URL into the browser history when the AJAX request completes.
	AttrHxPushUrl = MakePairAttribute("hx-push-url")
	// AttrHxVals sends additional values (JSON or JavaScript expression) with the AJAX request.
	AttrHxVals = MakePairAttribute("hx-vals")
)

// AttrHxOn dynamically creates an inline event listener attribute using the htmx syntax.
// It maps the specified event (e.g., "click", "htmx:config:request") to a JavaScript expression string.
//
// Example: AttrHxOn("click", "alert('Hello!')") becomes hx-on:click="alert('Hello!')"
func AttrHxOn(event, value string) PairAttribute {
	return PairAttribute{Key: "hx-on:" + event, Value: value}
}

const (
	SwapInnerHtml   = "innerHTML"   // Replaces inner content (Default)
	SwapOuterHtml   = "outerHTML"   // Replaces entire element
	SwapTextContent = "textContent" // Sets text content without parsing HTML
	SwapBeforeBegin = "beforebegin" // Inserts before target element
	SwapAfterBegin  = "afterbegin"  // Inserts as first child
	SwapBeforeEnd   = "beforeend"   // Inserts as last child
	SwapAfterEnd    = "afterend"    // Inserts after target element
	SwapPrepend     = "prepend"     // Same as [SwapAfterBegin]
	SwapAppend      = "append"      // Same as [SwapBeforeEnd]
	SwapDelete      = "delete"      // Removes target element completely
	SwapNone        = "none"        // Does not swap any content
	SwapInnerMorph  = "innerMorph"  // Morphs inner content, preserves state
	SwapOuterMorph  = "outerMorph"  // Morphs entire element, preserves state
	SwapOuterSync   = "outerSync"   // Merges attributes, replaces children
)

const (
	// EventHtmxConfigRequest is fired before an AJAX request is formulated.
	// This is the ideal place to dynamically modify request parameters, headers, or targets.
	EventHtmxConfigRequest = "htmx:config:request"
	// EventHtmxBeforeRequest is fired immediately before an AJAX request is dispatched over the network.
	// Can be canceled to prevent the request from going out.
	EventHtmxBeforeRequest = "htmx:before:request"
	// EventHtmxAfterRequest is fired after an AJAX request has finished,
	// regardless of whether it succeeded or failed (e.g., network errors, 404s).
	EventHtmxAfterRequest = "htmx:after:request"
	// EventHtmxFinallyRequest is fired as the final step of a request cycle,
	// after all processing, swapping, and cleaning up have completed. Perfect for teardown tasks.
	EventHtmxFinallyRequest = "htmx:finally:request"
	// EventHtmxBeforeResponse is fired right after a response is received from the server,
	// but before any processing or DOM swapping takes place.
	EventHtmxBeforeResponse = "htmx:before:response"
	// EventHtmxBeforeSwap is fired after a valid response is processed, but immediately
	// before the response HTML is actually injected/swapped into the DOM.
	EventHtmxBeforeSwap = "htmx:before:swap"
	// EventHtmxAfterSwap is fired immediately after the response HTML has been injected into the DOM.
	EventHtmxAfterSwap = "htmx:after:swap"
	// EventHtmxResponseError is fired when a remote server returns an error HTTP status code
	// (like a 400 or 500 error) instead of a successful hypermedia response.
	EventHtmxResponseError = "htmx:response:error"
	// EventHtmxError is fired when a general error occurs, such as a complete network
	// disconnection or a malformed request lifecycle error.
	EventHtmxError = "htmx:error"
	// EventHtmxConfirm is fired when htmx triggers an action that requests verification.
	// This allows you to hook up custom async modal confirmation dialogs.
	EventHtmxConfirm = "htmx:confirm"
	// EventHtmxBeforeProcess is fired before htmx parses an HTML element for hx-* attributes.
	// Essential for template engines or extensions to pre-populate elements dynamically.
	EventHtmxBeforeProcess = "htmx:before:process"
	// EventHtmxAfterProcess is fired after htmx finishes initializing an element and all
	// of its child nodes, making it safe to interact with via JavaScript.
	EventHtmxAfterProcess = "htmx:after:process"
)

// HxPartial creates a new htmx 4 multi-target partial element (<hx-partial>).
//
// In htmx 4, <hx-partial> tags are returned in a single server response to target
// and swap multiple distinct elements across the DOM simultaneously. They act as
// a cleaner, explicit alternative to legacy out-of-band (hx-swap-oob) swaps.
//
// Inside the slice of Attributes, you should specify the target elements and swap styles
// explicitly for each specific partial block.
func HxPartial(attrs ...Attribute) ChildrenInserter {
	return MakeChildrenInserter(Element{Tag: "hx-partial", Attributes: attrs})
}
