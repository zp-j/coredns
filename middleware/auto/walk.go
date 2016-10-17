package auto

import (
	"log"
	"os"
	"path"
	"path/filepath"
	"regexp"

	"github.com/miekg/coredns/middleware/file"

	"github.com/miekg/dns"
)

// Walk will recursively walk of the file under l.directory and adds the one that match l.re.
func (z *Zones) Walk(l loader) error {

	// TODO(miek): should add something so that we don't stomp on each other.

	toDelete := make(map[string]bool)
	for _, n := range z.Names() {
		toDelete[n] = true
	}

	filepath.Walk(l.directory, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}

		match, origin := matches(l.re, info.Name(), l.template)
		if !match {
			return nil
		}

		if _, ok := z.Z[origin]; ok {
			// we already have this zone
			toDelete[origin] = false
			return nil
		}

		reader, err := os.Open(path)
		if err != nil {
			log.Printf("[WARNING] Opening %s failed: %s", path, err)
			return nil
		}

		zo, err := file.Parse(reader, origin, path)
		if err != nil {
			// Parse barfs warning by itself...
			return nil
		}

		zo.NoReload = l.noReload
		zo.TransferTo = l.transferTo

		z.Insert(zo, origin)

		zo.Notify()

		log.Printf("[INFO] Inserting zone `%s' from: %s", origin, path)

		toDelete[origin] = false

		return nil
	})

	for origin, ok := range toDelete {
		if !ok {
			continue
		}
		z.Delete(origin)
		log.Printf("[INFO] Deleting zone `%s'", origin)
	}

	return nil
}

// matches matches re to filename, if is is a match, the subexpression will be used to expand
// template to an origin. When match is true that origin is returned. Origin is fully qualified.
func matches(re *regexp.Regexp, filename, template string) (match bool, origin string) {
	base := path.Base(filename)

	matches := re.FindStringSubmatchIndex(base)
	if matches == nil {
		return false, ""
	}

	by := re.ExpandString(nil, template, base, matches)
	if by == nil {
		return false, ""
	}

	origin = dns.Fqdn(string(by))

	return true, origin
}
