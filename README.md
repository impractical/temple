# temple

temple is an HTML templating library for Go.
It doesn't use its own syntax, preferring to use the standard library's [html/template](https://pkg.go.dev/html/template)s.
temple's purpose is to make working with the standard library's templates a little bit more manageable.

It is for people who want to render HTML from their servers like it's 2003 again.

## Approach

### Component-driven

Go templates come in two parts: the template literal, and the data to render that template literal with.
These two parts are tied together in a fragile protocol that the compiler does not help you at all with.
You need to make sure that the data to render is passed in the format the template literal expects, and that when one changes the other changes, as well.

temple addresses this by wrapping both parts inside an abstraction called a "Component".
A [Component](https://pkg.go.dev/impractical.co/temple#Component) is anything that can surface a template literal.

A Component that can be rendered as a standalone page, instead of just being part of another page, is called a [Page](https://pkg.go.dev/impractical.co/temple#Page).
When a Page is rendered the data passed to the template literal will contain the Page itself as `$.Page`.
In this way, temple codifies the data being passed in, and identifies the template literal it is passed to at the same time.
A Page can serve as the interface that a template is accessed through, exposing the data it expects and the format it expects it in.
A Component allows a Page to reference another template through an interface that exposes the data it expects and the format it expects it in.

### Access to global state

It's a pain to try and build everything out of Components that you explicitly pass state through.
Sometimes you want something like your site's name to be configurable, and you don't want to pipe that through every Component.
To that end, temple defines [a "Site" type](https://pkg.go.dev/impractical.co/temple#Site).
The Site will be included in the data passed to every template, as `$.Site`.

### JavaScript and CSS stay with their Components

A Component can optionally indicate that it [embeds JavaScript directly into the HTML](https://pkg.go.dev/impractical.co/temple#JSEmbedder), [links to a JavaScript file loaded at runtime](https://pkg.go.dev/impractical.co/temple#JSLinker), [embeds CSS directly into the HTML](https://pkg.go.dev/impractical.co/temple#CSSEmbedder), [links to a CSS file loaded at runtime](https://pkg.go.dev/impractical.co/temple#CSSLinker), or any combination of these approaches.
JavaScript and CSS can get loaded only on the pages they are needed for by tagging along with Components when the Component is rendered.

Any of these resources can also declare a relationship to another resource, controlling the order in which they get rendered to the page, allowing for fine-grained control over how resources end up being loaded in the HTML.
By default, if any Component's Embedder or Linker methods return more than one resource in a single slice, those resources will be rendered in the order they appear in the slice.
Explicitly declaring a relationship to any other resources disables this implicit relationship, as does setting the `DisableImplicitOrdering` property on the resource to `true`.

### HTML agnostic

temple is a layer on top of `html/template`, but it isn't attempting to proscribe how you write your HTML.
It strives to offer the least-restrictive possible interface on top of `html/template`; just enough to create its Components system and resource-loading system.

## Examples

See the [runnable, tested examples on pkg.go.dev](https://pkg.go.dev/impractical.co/temple#Render).
