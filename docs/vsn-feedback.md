# VSN development feedback

Status: Step 6 implementation record for the phase-one preview interaction.

## Version and artifact

The implementation was developed against the local VSN repository at
`/Users/mattroberts/Projects/vsn`:

- package version: `1.0.15`
- source revision: `32c7e21`
- browser entry point: `dist/index.min.js`
- vendored artifact size: `167109` bytes
- vendored artifact SHA-256: `7e5be5cfb72820bcd683788b2cae969de930b4bf5022f09a987c5d1deb473d2b`

The source and its examples were inspected directly. The usable API for this
project is the current `Engine`/`autoMount` runtime with `vsn-get`,
`vsn-target`, and `vsn-swap` attributes.

## Interaction implemented

Each read-only list row has a normal detail `href` and a VSN enhancement:

```html
<a href="/control/books/42"
   vsn-get="/control/books/42"
   vsn-target="#record-preview"
   vsn-swap="inner">Preview</a>
```

With VSN loaded, the click is fetched and the returned detail fragment is
swapped into `#record-preview`. Without JavaScript, the same anchor navigates
to `/control/books/42`, which returns the complete server-rendered detail page.
The server distinguishes the enhanced request using VSN's verified
`HX-Request: true` header; ordinary requests do not receive a fragment.

The VSN module is vendored and served as
`<base-path>/assets/vsn.min.js`, so the interaction works at `/`, `/admin`,
`/control`, or another configured mount path without a CDN dependency.

## What worked well

- VSN starts from existing HTML and does not require a client-side page
  renderer or a second component system.
- `vsn-get` plus an explicit target and swap maps directly to a small
  server-rendered fragment workflow.
- Keeping the ordinary `href` made the fallback and direct-link behavior
  straightforward.
- `autoMount` lets the page load the runtime without application-specific
  JavaScript bootstrap code.

## Friction and awkward edges

- Templ's attribute parser rejected the modifier spelling `vsn-get!trusted`,
  even though the VSN HTML examples use modifier names. Plain `vsn-get` is
  sufficient for this server-rendered fragment and keeps VSN's default
  HTML-safety path active.
- VSN's current request contract uses `HX-*` headers, including `HX-Request`
  and `HX-Target`; the earlier discovery notes referring to `X-Requested-With`
  and legacy `vsn-xhr` names do not match the local VSN source used for this
  implementation.
- VSN is a browser asset rather than a Go module. A Go package that wants a
  self-contained example must either vendor a pinned build, require a host
  asset pipeline, or depend on a CDN. This implementation chose a pinned
  vendored build so the local example has no external runtime dependency.
- The enhanced path is asynchronous and browser-side; Go handler tests can
  verify the emitted attributes, partial-request contract, and fallback, but
  browser-level interaction tests belong in a future example/e2e harness.

## Follow-up ideas for VSN

- Publish a small, stable integration note covering the request headers,
  `autoMount` asset loading, trusted/sanitized fragment choices, and fallback
  expectations.
- Consider a documented way for server-rendered template systems to express
  modifier-bearing custom attributes such as `vsn-get!trusted` without raw
  HTML escape hatches.
- Provide versioned browser artifacts or an explicit embedding/build recipe
  so Go packages can pin the runtime without copying a large minified file by
  hand.
