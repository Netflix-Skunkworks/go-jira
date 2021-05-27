package jiracli

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"time"

	kingpin "gopkg.in/alecthomas/kingpin.v2"

	"github.com/coryb/figtree"
)

func Homedir() string {
	if runtime.GOOS == "windows" {
		return os.Getenv("USERPROFILE")
	}
	return os.Getenv("HOME")
}

func Cookiedir() string {
    value, exists := os.LookupEnv("XDG_RUNTIME_DIR")
    if !exists {
        value = Homedir()
    }
    return value
}

func findClosestParentPath(fileName string) (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	paths := figtree.FindParentPaths(Homedir(), cwd, fileName)
	if len(paths) > 0 {
		return paths[len(paths)-1], nil
	}
	return "", fmt.Errorf("%s not found in parent directory hierarchy", fileName)
}

func tmpYml(tmpFilePrefix string) (*os.File, error) {
	fh, err := ioutil.TempFile("", filepath.Base(tmpFilePrefix))
	if err != nil {
		return nil, err
	}
	// now we need to rename the file since we dont control the file extensions
	// ... it has to be `.yml` so that vim/emacs etc know what edit mode to apply
	// for easier editing
	oldFileName := fh.Name()
	newFileName := oldFileName + ".yml"

	// close tmpfile so we can rename on windows
	fh.Close()

	if err := os.Rename(oldFileName, newFileName); err != nil {
		return nil, err
	}

	return os.OpenFile(newFileName, os.O_RDWR|os.O_EXCL, 0600)
}

func FlagValue(ctx *kingpin.ParseContext, name string) string {
	for _, elem := range ctx.Elements {
		if flag, ok := elem.Clause.(*kingpin.FlagClause); ok {
			if flag.Model().Name == name {
				return *elem.Value
			}
		}
	}
	return ""
}

func copyFile(src, dst string) (err error) {
	var s, d *os.File
	if s, err = os.Open(src); err == nil {
		defer s.Close()
		if d, err = os.Create(dst); err == nil {
			if _, err = io.Copy(d, s); err != nil {
				d.Close()
				return
			}
			return d.Close()
		}
	}
	return
}

func fuzzyAge(start string) (string, error) {
	t, err := time.Parse("2006-01-02T15:04:05.000-0700", start)
	if err != nil {
		return "", err
	}
	delta := time.Since(t)
	if delta.Minutes() < 2 {
		return "a minute", nil
	} else if dm := delta.Minutes(); dm < 45 {
		return fmt.Sprintf("%d minutes", int(dm)), nil
	} else if dm := delta.Minutes(); dm < 90 {
		return "an hour", nil
	} else if dh := delta.Hours(); dh < 24 {
		return fmt.Sprintf("%d hours", int(dh)), nil
	} else if dh := delta.Hours(); dh < 48 {
		return "a day", nil
	}
	return fmt.Sprintf("%d days", int(delta.Hours()/24)), nil
}

func dateFormat(format string, content string) (string, error) {
	t, err := time.Parse("2006-01-02T15:04:05.000-0700", content)
	if err != nil {
		return "", err
	}
	return t.Format(format), nil
}

// this is a HACK to make yaml parsed documents to be serializable
// to json, so prevent this:
// json: unsupported type: map[interface {}]interface {}
// Also we want to clean up common input errors for the edit
// templates, like dangling "\n"
func yamlFixup(data interface{}) (interface{}, error) {
	switch d := data.(type) {
	case map[interface{}]interface{}:
		// need to copy this map into a string map so json can encode it
		copy := make(map[string]interface{})
		for key, val := range d {
			switch k := key.(type) {
			case string:
				if fixed, err := yamlFixup(val); err != nil {
					return nil, err
				} else if fixed != nil {
					copy[k] = fixed
				}
			default:
				err := fmt.Errorf("YAML: key %s is type '%T', require 'string'", key, k)
				log.Errorf("%s", err)
				return nil, err
			}
		}
		if len(copy) == 0 {
			return nil, nil
		}
		return copy, nil
	case map[string]interface{}:
		copy := make(map[string]interface{})
		for k, v := range d {
			if fixed, err := yamlFixup(v); err != nil {
				return nil, err
			} else if fixed != nil {
				copy[k] = fixed
			}
		}
		if len(copy) == 0 {
			return nil, nil
		}
		return copy, nil
	case []interface{}:
		copy := make([]interface{}, 0, len(d))
		for _, val := range d {
			if fixed, err := yamlFixup(val); err != nil {
				return nil, err
			} else if fixed != nil {
				copy = append(copy, fixed)
			}
		}
		if len(copy) == 0 {
			return nil, nil
		}
		return copy, nil
	case *interface{}:
		if fixed, err := yamlFixup(*d); err != nil {
			return nil, err
		} else if fixed != nil {
			*d = fixed
		}
		return d, nil
	case string:
		if d == "" || d == "\n" {
			return nil, nil
		}
		return d, nil
	default:
		return d, nil
	}
}
