package protocolgate

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

type npmLock struct {
	Name            string                    `json:"name"`
	LockfileVersion int                       `json:"lockfileVersion"`
	Packages        map[string]npmLockPackage `json:"packages"`
}

type npmLockPackage struct {
	Version      string            `json:"version"`
	Resolved     string            `json:"resolved"`
	Integrity    string            `json:"integrity"`
	Dependencies map[string]string `json:"dependencies"`
}

func VerifyNPMLock(path string, pins Pins) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	var lock npmLock
	if err := decoder.Decode(&lock); err != nil {
		return fmt.Errorf("decode npm lock: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("decode npm lock: trailing JSON value")
		}
		return fmt.Errorf("decode npm lock trailing data: %w", err)
	}
	if lock.Name != "jetkvm-mcp-protocol-gates" || lock.LockfileVersion != 3 {
		return errors.New("npm lock identity or format differs from the reviewed contract")
	}
	root, ok := lock.Packages[""]
	if !ok || len(root.Dependencies) != 2 || root.Dependencies[pins.Conformance.Package] != pins.Conformance.Version || root.Dependencies[pins.Inspector.Package] != pins.Inspector.Version {
		return errors.New("npm lock root dependencies differ from the reviewed pins")
	}
	if err := verifyPinnedNPMPackage(lock.Packages, pins.Conformance.Package, pins.Conformance.Version, pins.Conformance.Integrity); err != nil {
		return fmt.Errorf("conformance lock entry: %w", err)
	}
	if err := verifyPinnedNPMPackage(lock.Packages, pins.Inspector.Package, pins.Inspector.Version, pins.Inspector.Integrity); err != nil {
		return fmt.Errorf("inspector lock entry: %w", err)
	}
	for path, pkg := range lock.Packages {
		if path == "" {
			continue
		}
		if pkg.Version == "" || !strings.HasPrefix(pkg.Resolved, "https://registry.npmjs.org/") || !integrityPattern.MatchString(pkg.Integrity) {
			return fmt.Errorf("npm lock package %q is not an immutable registry artifact", path)
		}
	}
	return nil
}

func verifyPinnedNPMPackage(packages map[string]npmLockPackage, packageName, version, integrity string) error {
	pkg, ok := packages["node_modules/"+packageName]
	if !ok {
		return errors.New("package is absent")
	}
	if pkg.Version != version || pkg.Integrity != integrity {
		return errors.New("version or integrity differs from the reviewed source pin")
	}
	return nil
}
