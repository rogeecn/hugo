// Copyright 2021 The Hugo Authors. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package paths

import (
	"fmt"
	"net/url"
	"path"
	"strings"

	"github.com/PuerkitoBio/purell"
)

type pathBridge struct {
}

func (pathBridge) Base(in string) string {
	return path.Base(in)
}

func (pathBridge) Clean(in string) string {
	return path.Clean(in)
}

func (pathBridge) Dir(in string) string {
	return path.Dir(in)
}

func (pathBridge) Ext(in string) string {
	return path.Ext(in)
}

func (pathBridge) Join(elem ...string) string {
	return path.Join(elem...)
}

func (pathBridge) Separator() string {
	return "/"
}

var pb pathBridge

func sanitizeURLWithFlags(in string, f purell.NormalizationFlags) string {
	s, err := purell.NormalizeURLString(in, f)
	if err != nil {
		return in
	}

	// Temporary workaround for the bug fix and resulting
	// behavioral change in purell.NormalizeURLString():
	// a leading '/' was inadvertently added to relative links,
	// but no longer, see #878.
	//
	// I think the real solution is to allow Hugo to
	// make relative URL with relative path,
	// e.g. "../../post/hello-again/", as wished by users
	// in issues #157, #622, etc., without forcing
	// relative URLs to begin with '/'.
	// Once the fixes are in, let's remove this kludge
	// and restore SanitizeURL() to the way it was.
	//                         -- @anthonyfok, 2015-02-16
	//
	// Begin temporary kludge
	u, err := url.Parse(s)
	if err != nil {
		panic(err)
	}
	if len(u.Path) > 0 && !strings.HasPrefix(u.Path, "/") {
		u.Path = "/" + u.Path
	}
	return u.String()
	// End temporary kludge

	// return s

}

// SanitizeURL sanitizes the input URL string.
func SanitizeURL(in string) string {
	return sanitizeURLWithFlags(in, purell.FlagsSafe|purell.FlagRemoveTrailingSlash|purell.FlagRemoveDotSegments|purell.FlagRemoveDuplicateSlashes|purell.FlagRemoveUnnecessaryHostDots|purell.FlagRemoveEmptyPortSeparator)
}

// SanitizeURLKeepTrailingSlash is the same as SanitizeURL, but will keep any trailing slash.
func SanitizeURLKeepTrailingSlash(in string) string {
	return sanitizeURLWithFlags(in, purell.FlagsSafe|purell.FlagRemoveDotSegments|purell.FlagRemoveDuplicateSlashes|purell.FlagRemoveUnnecessaryHostDots|purell.FlagRemoveEmptyPortSeparator)
}

// MakePermalink combines base URL with content path to create full URL paths.
// Example
//    base:   http://spf13.com/
//    path:   post/how-i-blog
//    result: http://spf13.com/post/how-i-blog
func MakePermalink(host, plink string) *url.URL {
	base, err := url.Parse(host)
	if err != nil {
		panic(err)
	}

	p, err := url.Parse(plink)
	if err != nil {
		panic(err)
	}

	if p.Host != "" {
		panic(fmt.Errorf("can't make permalink from absolute link %q", plink))
	}

	base.Path = path.Join(base.Path, p.Path)

	// path.Join will strip off the last /, so put it back if it was there.
	hadTrailingSlash := (plink == "" && strings.HasSuffix(host, "/")) || strings.HasSuffix(p.Path, "/")
	if hadTrailingSlash && !strings.HasSuffix(base.Path, "/") {
		base.Path = base.Path + "/"
	}

	return base
}

// IsAbsURL determines whether the given path points to an absolute URL.
func IsAbsURL(path string) bool {
	url, err := url.Parse(path)
	if err != nil {
		return false
	}

	return url.IsAbs() || strings.HasPrefix(path, "//")
}

// AddContextRoot adds the context root to an URL if it's not already set.
// For relative URL entries on sites with a base url with a context root set (i.e. http://example.com/mysite),
// relative URLs must not include the context root if canonifyURLs is enabled. But if it's disabled, it must be set.
func AddContextRoot(baseURL, relativePath string) string {
	url, err := url.Parse(baseURL)
	if err != nil {
		panic(err)
	}

	newPath := path.Join(url.Path, relativePath)

	// path strips trailing slash, ignore root path.
	if newPath != "/" && strings.HasSuffix(relativePath, "/") {
		newPath += "/"
	}
	return newPath
}

// URLizeAn

// PrettifyURL takes a URL string and returns a semantic, clean URL.
func PrettifyURL(in string) string {
	x := PrettifyURLPath(in)

	if path.Base(x) == "index.html" {
		return path.Dir(x)
	}

	if in == "" {
		return "/"
	}

	return x
}

// PrettifyURLPath takes a URL path to a content and converts it
// to enable pretty URLs.
//     /section/name.html       becomes /section/name/index.html
//     /section/name/           becomes /section/name/index.html
//     /section/name/index.html becomes /section/name/index.html
func PrettifyURLPath(in string) string {
	return prettifyPath(in, pb)
}

// Uglify does the opposite of PrettifyURLPath().
//     /section/name/index.html becomes /section/name.html
//     /section/name/           becomes /section/name.html
//     /section/name.html       becomes /section/name.html
func Uglify(in string) string {
	if path.Ext(in) == "" {
		if len(in) < 2 {
			return "/"
		}
		// /section/name/  -> /section/name.html
		return path.Clean(in) + ".html"
	}

	name, ext := fileAndExt(in, pb)
	if name == "index" {
		// /section/name/index.html -> /section/name.html
		d := path.Dir(in)
		if len(d) > 1 {
			return d + ext
		}
		return in
	}
	// /.xml -> /index.xml
	if name == "" {
		return path.Dir(in) + "index" + ext
	}
	// /section/name.html -> /section/name.html
	return path.Clean(in)
}
