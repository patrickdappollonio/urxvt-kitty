package main

import (
	"bytes"
	"errors"
	"fmt"
	"image/color"
	"io"
	"net/url"
	"os"
	"regexp"
	"sort"
	"strings"
)

var reParseItems = regexp.MustCompile(`\*\.(color[0-9]{1,2}|foreground|background|cursorColor)+: +(#[a-fA-F0-9]{6})`)

const colorPrefix = "Colour"

var nameReplacements = map[string][]int{
	"foreground":  {0, 1},
	"background":  {2, 3},
	"cursorColor": {4, 5},
	"color0":      {6},
	"color8":      {7},
	"color1":      {8},
	"color9":      {9},
	"color2":      {10},
	"color10":     {11},
	"color3":      {12},
	"color11":     {13},
	"color4":      {14},
	"color12":     {15},
	"color5":      {16},
	"color13":     {17},
	"color6":      {18},
	"color14":     {19},
	"color7":      {20},
	"color15":     {21},
}

type colormatch struct {
	name  string
	color color.RGBA
}

func (cm *colormatch) getRGB() string {
	return fmt.Sprintf("%d,%d,%d", cm.color.R, cm.color.G, cm.color.B)
}

func main() {
	if err := app(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err.Error())
		os.Exit(1)
	}
}

func app() error {
	switch len(os.Args[1:]) {
	case 2:
		// do nothing, we'll handle below
	default:
		return errors.New("usage: urxvt-kitty [filename] [sessionName] -- get colors from: http://dotshare.it/category/terms/colors/")
	}

	fname, sname := os.Args[1], os.Args[2]

	if sname == "" {
		return errors.New("session name is empty")
	}

	f, err := os.Open(fname)
	if err != nil {
		return fmt.Errorf("can't open file %q: %s", fname, err.Error())
	}

	defer f.Close()

	var b bytes.Buffer
	if _, err := io.Copy(&b, f); err != nil {
		return fmt.Errorf("can't read file %q: %s", fname, err.Error())
	}

	objects := reParseItems.FindAllStringSubmatch(b.String(), -1)
	if len(objects) == 0 {
		return fmt.Errorf("file %q format is invalid: no color codes found", fname)
	}

	values := map[string]string{}
	for idx, v := range objects {
		if len(v) != 3 {
			return fmt.Errorf("no color code format found in mapping submatch at position %d: mappings: %#v", idx, v)
		}

		values[v[1]] = v[2]
	}

	notFoundKeys := make([]string, 0, len(values))
	kvals := make([]colormatch, 0, len(nameReplacements)+3)

	for keyName, keyItems := range nameReplacements {
		hexColor, found := values[keyName]

		if !found {
			notFoundKeys = append(notFoundKeys, keyName)
			continue
		}

		converted, err := hexToRGB(hexColor)
		if err != nil {
			return fmt.Errorf("unable to parse hex color %q: %s", hexColor, err.Error())
		}

		for _, m := range keyItems {
			kvals = append(kvals, colormatch{
				name:  fmt.Sprintf("%s%d", colorPrefix, m),
				color: converted,
			})
		}
	}

	if len(notFoundKeys) != 0 {
		return fmt.Errorf("the following keys weren't found in the config file: %s", strings.Join(notFoundKeys, ", "))
	}

	sort.Slice(kvals, func(i, j int) bool {
		return kvals[i].name < kvals[j].name
	})

	b.Reset()

	fmt.Fprintln(&b, "Windows Registry Editor Version 5.00")
	fmt.Fprintln(&b, "")
	fmt.Fprintf(&b, "[HKEY_CURRENT_USER\\Software\\9bis.com\\KiTTY\\Sessions\\%s]\n", url.PathEscape(sname))

	for _, color := range kvals {
		fmt.Fprintf(&b, "%q=%q\n", color.name, color.getRGB())
	}

	fmt.Fprintln(os.Stdout, b.String())

	return nil
}

func hexToRGB(s string) (c color.RGBA, err error) {
	c.A = 0xff

	if s[0] != '#' {
		return c, errors.New("invalid format")
	}

	hexToByte := func(b byte) byte {
		switch {
		case b >= '0' && b <= '9':
			return b - '0'
		case b >= 'a' && b <= 'f':
			return b - 'a' + 10
		case b >= 'A' && b <= 'F':
			return b - 'A' + 10
		}
		err = errors.New("invalid format")
		return 0
	}

	switch len(s) {
	case 7:
		c.R = hexToByte(s[1])<<4 + hexToByte(s[2])
		c.G = hexToByte(s[3])<<4 + hexToByte(s[4])
		c.B = hexToByte(s[5])<<4 + hexToByte(s[6])
	case 4:
		c.R = hexToByte(s[1]) * 17
		c.G = hexToByte(s[2]) * 17
		c.B = hexToByte(s[3]) * 17
	default:
		err = errors.New("invalid format")
	}
	return
}
