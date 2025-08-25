// Package temple provides an HTML rendering framework built on top of the
// html/template package.
//
// temple is organized around Components and Pages. A Component is some piece
// of the HTML document that you want included in the page's output. A Page is
// a Component that gets rendered itself rather than being included in another
// Component. The homepage of a site is probably a Page; the site's navbar is
// probably a Component, as is the base layout that all pages have in common.
//
// temple also has the concept of a Site. Each server should have a Site, which
// acts as a singleton for the server and provides the fs.FS containing the
// templates that Components are using. A Site will also be available at render
// time, as .Site, so it can hold configuration data used across all pages.
//
// To render a page, pass it to the Render function. The page itself will be
// made available as .Page within the template, and the Site will be available
// as .Site.
//
// Components tend to be structs, with properties for whatever data they want
// to pass to their templates. When a Component relies on another Component,
// our homepage including a navbar for example, a good practice is to make an
// instance of the navbar Component a property on the homepage Component
// struct. That allows the homepage to select, e.g., which link in the navbar
// is highlighted as active. It's also a good idea to include the navbar
// Component in the output of a UseComponents method on the homepage Component,
// so all its methods (the templates it uses, any CSS or JS that it embeds or
// links to, any Components _it_ relies on...) will all get included whenever
// the homepage Component is rendered.
package temple
