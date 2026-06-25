package templates

import . "github.com/assaidy/hyper/v2"

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

var (
	AttrHxGet        = MakePairAttribute("hx-get")         // Issues a GET request to the specified URL
	AttrHxPost       = MakePairAttribute("hx-post")        // Issues a POST request to the specified URL
	AttrHxPut        = MakePairAttribute("hx-put")         // Issues a PUT request to the specified URL
	AttrHxPatch      = MakePairAttribute("hx-patch")       // Issues a PATCH request to the specified URL
	AttrHxDelete     = MakePairAttribute("hx-delete")      // Issues a DELETE request to the specified URL
	AttrHxTrigger    = MakePairAttribute("hx-trigger")     // Controls when the element issues a request
	AttrHxSwap       = MakePairAttribute("hx-swap")        // Controls how the response is inserted
	AttrHxTarget     = MakePairAttribute("hx-target")      // Controls where the response is inserted
	AttrHxSelect     = MakePairAttribute("hx-select")      // Controls which part of the response to insert
	AttrHxSwapOob    = MakePairAttribute("hx-swap-oob")    // Marks response elements to swap into the page by ID
	AttrHxSelectOob  = MakePairAttribute("hx-select-oob")  // Picks response elements to swap into the page by ID
	AttrHxConfirm    = MakePairAttribute("hx-confirm")     // Shows a confirmation dialog before the request
	AttrHxOn         = MakePairAttribute("hx-on")          // Runs inline JavaScript when an event fires
	AttrHxVals       = MakePairAttribute("hx-vals")        // Adds values to request parameters
	AttrHxInclude    = MakePairAttribute("hx-include")     // Includes additional element values in the request
	AttrHxHeaders    = MakePairAttribute("hx-headers")     // Adds custom headers to the request
	AttrHxEncoding   = MakePairAttribute("hx-encoding")    // Sets the request encoding type
	AttrHxPushUrl    = MakePairAttribute("hx-push-url")    // Pushes the URL into browser history
	AttrHxReplaceUrl = MakePairAttribute("hx-replace-url") // Replaces the current URL in browser history
	AttrHxHistoryElt = MakePairAttribute("hx-history-elt") // Marks the element to swap on history restore
	AttrHxBoost      = MakePairAttribute("hx-boost")       // Converts links and forms to AJAX
	AttrHxPreload    = MakePairAttribute("hx-preload")     // Preloads content before the user triggers a request
	AttrHxOptimistic = MakePairAttribute("hx-optimistic")  // Shows optimistic content during the request
	AttrHxIndicator  = MakePairAttribute("hx-indicator")   // Specifies the loading indicator element
	AttrHxStatus     = MakePairAttribute("hx-status")      // Handles responses differently by status code
	AttrHxSync       = MakePairAttribute("hx-sync")        // Synchronizes requests between elements
	AttrHxValidate   = MakePairAttribute("hx-validate")    // Validates before submitting the request
	AttrHxDisable    = MakePairAttribute("hx-disable")     // Disables elements during the request
	AttrHxIgnore     = MakePairAttribute("hx-ignore")      // Disables htmx processing for the element
	AttrHxPreserve   = MakePairAttribute("hx-preserve")    // Preserves the element during swaps
	AttrHxAction     = MakePairAttribute("hx-action")      // Specifies the URL to receive the request
	AttrHxMethod     = MakePairAttribute("hx-method")      // Specifies the HTTP method for the request
	AttrHxConfig     = MakePairAttribute("hx-config")      // Configures request behavior with JSON
)

// AttrHxOnEvent Runs inline JavaScript when an event fires.
// It maps the specified event (e.g., "click", "htmx:config:request") to a JavaScript expression string.
//
// Example: AttrHxOnEvent("click", "alert('Hello!')") becomes hx-on:click="alert('Hello!')"
func AttrHxOnEvent(event, value string) Attribute {
	return PairAttribute{Key: "hx-on:" + event, Value: value}
}

const (
	SwapInnerHtml   = "innerHTML"   // Replaces inner content (Default)
	SwapOuterHtml   = "outerHTML"   // Replaces entire element
	SwapTextContent = "textContent" // Sets text content without parsing HTML
	SwapBeforeBegin = "beforebegin" // Inserts before target element
	SwapBefore      = "before"      // Same as [SwapBeforeBegin]
	SwapAfterBegin  = "afterbegin"  // Inserts as first child
	SwapPrepend     = "prepend"     // Same as [SwapAfterBegin]
	SwapBeforeEnd   = "beforeend"   // Inserts as last child
	SwapAppend      = "append"      // Same as [SwapBeforeEnd]
	SwapAfterEnd    = "afterend"    // Inserts after target element
	SwapAfter       = "after"       // Same as [SwapAfterEnd]
	SwapDelete      = "delete"      // Removes target element completely
	SwapNone        = "none"        // Does not swap any content
	SwapInnerMorph  = "innerMorph"  // Morphs inner content, preserves state
	SwapOuterMorph  = "outerMorph"  // Morphs entire element, preserves state
	SwapOuterSync   = "outerSync"   // Merges attributes, replaces children
	SwapUpsert      = "upsert"      // Updates existing elements by ID and inserts new ones. Requires the upsert extension.
)

const (
	HeaderHxRequest              = "HX-Request"                 // Indicates a request was made by htmx
	HeaderHxRequestType          = "HX-Request-Type"            // Indicates if this is a partial or full page request
	HeaderHxCurrentUrl           = "HX-Current-URL"             // Contains the URL of the browser when the request was made
	HeaderHxSource               = "HX-Source"                  // Identifies the element that triggered the request
	HeaderHxTarget               = "HX-Target"                  // The element that will receive the response
	HeaderHxBoosted              = "HX-Boosted"                 // Indicates a boosted navigation request
	HeaderHxHistoryRestoreReques = "HX-History-Restore-Request" // Indicates history navigation (back/forward)
	HeaderAccept                 = "Accept"                     // Content types htmx accepts from the server
	HeaderLastEventId            = "Last-Event-ID"              // Last received SSE event ID for reconnection
	HeaderHxTrigger              = "HX-Trigger"                 // Trigger client-side events from the server
	HeaderHxLocation             = "HX-Location"                // Client-side AJAX navigation to a new URL
	HeaderHxRedirect             = "HX-Redirect"                // Client-side redirect to a new URL
	HeaderHxRefresh              = "HX-Refresh"                 // Trigger a full page reload
	HeaderHxRetarget             = "HX-Retarget"                // Override the swap target from the server
	HeaderHxReswap               = "HX-Reswap"                  // Override the swap style from the server
	HeaderHxReselect             = "HX-Reselect"                // Override the content selection from the server
	HeaderHxReplaceUrl           = "HX-Replace-Url"             // Replace the current URL in the browser history
	HeaderHxPushUrl              = "HX-Push-Url"                // Push a URL into the browser history stack
)

const (
	EventHtmxConfigRequest        = "htmx:config:request"            // Configure request before it's sent
	EventHtmxBeforeRequest        = "htmx:before:request"            // Immediately before fetch is called
	EventHtmxAfterRequest         = "htmx:after:request"             // After response is received
	EventHtmxFinallyRequest       = "htmx:finally:request"           // At the end of request lifecycle
	EventHtmxBeforeSwap           = "htmx:before:swap"               // Before content is swapped into DOM
	EventHtmxAfterSwap            = "htmx:after:swap"                // After content is swapped into DOM
	EventHtmxBeforeCleanup        = "htmx:before:cleanup"            // Before htmx removes element data
	EventHtmxAfterCleanup         = "htmx:after:cleanup"             // After listeners and data are removed
	EventHtmxConfirm              = "htmx:confirm"                   // Show confirmation dialog before request
	EventHtmxError                = "htmx:error"                     // When an error occurs during request
	EventHtmxAbort                = "htmx:abort"                     // Trigger to abort an ongoing request
	EventHtmxBeforeInit           = "htmx:before:init"               // Before a specific element is initialized
	EventHtmxAfterInit            = "htmx:after:init"                // After an element is fully initialized
	EventHtmxBeforeProcess        = "htmx:before:process"            // Before htmx processes a DOM node
	EventHtmxAfterProcess         = "htmx:after:process"             // After htmx processes a DOM node
	EventHtmxProcess              = "htmx:process"                   // Custom template processing (append :{type})
	EventHtmxAfterImplicitInherit = "htmx:after:implicitInheritance" // After implicit inheritance is applied
	EventHtmxBeforeHistoryUpdate  = "htmx:before:history:update"     // Before browser history is updated
	EventHtmxAfterHistoryUpdate   = "htmx:after:history:update"      // After browser history is updated
	EventHtmxAfterHistoryPush     = "htmx:after:history:push"        // After a push state action
	EventHtmxAfterHistoryReplace  = "htmx:after:history:replace"     // After a replace state action
	EventHtmxBeforeHistoryRestore = "htmx:before:history:restore"    // Before restoring from history
	EventHtmxBeforeViewTransition = "htmx:before:viewTransition"     // Before View Transition API starts
	EventHtmxAfterViewTransition  = "htmx:after:viewTransition"      // After View Transition completes
	EventLoad                     = "load"                           // Fired immediately after initialization
	EventIntersect                = "intersect"                      // Element enters viewport
	EventEvery                    = "every"                          // Periodic polling trigger
	EventHtmxBeforeResponse       = "htmx:before:response"           // After a response is received but before the body is consumed
	EventHtmxBeforeSettle         = "htmx:before:settle"             // Before the settle phase begins after a swap
	EventHtmxAfterSettle          = "htmx:after:settle"              // After the settle phase completes
	EventHtmxResponseError        = "htmx:response:error"            // When an HTTP error status (4xx/5xx) is received
)
