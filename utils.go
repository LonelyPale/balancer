package balancer

import (
	"bytes"
)

// URLSeparator define url separator
const URLSeparator = '/'

// URLJoin splicing url
func URLJoin(urls ...string) string {
	elems := make([]string, 0)
	for _, url := range urls {
		if len(url) > 0 {
			elems = append(elems, url)
		}
	}

	switch len(elems) {
	case 0:
		return ""
	case 1:
		return elems[0]
	}

	n := len(string(URLSeparator)) * (len(elems) - 1)
	for i := 0; i < len(elems); i++ {
		n += len(elems[i])
	}

	buffer := bytes.NewBufferString(elems[0])
	buffer.Grow(n)

	for i := 1; i < len(elems); i++ {
		last := elems[i-1]
		ln := len(last)

		if last[ln-1] == URLSeparator {
			if elems[i][0] == URLSeparator {
				if _, err := buffer.WriteString(elems[i][1:]); err != nil {
					panic(err)
				}
			} else {
				if _, err := buffer.WriteString(elems[i]); err != nil {
					panic(err)
				}
			}
		} else {
			if elems[i][0] == URLSeparator {
				if _, err := buffer.WriteString(elems[i]); err != nil {
					panic(err)
				}
			} else {
				if _, err := buffer.WriteString(string(URLSeparator)); err != nil {
					panic(err)
				}
				if _, err := buffer.WriteString(elems[i]); err != nil {
					panic(err)
				}
			}
		}
	}

	return buffer.String()
}

// HashCode generate hash code
func HashCode(key string) int {
	if len(key) == 0 {
		return 0
	}

	hash := 0
	chars := []byte(key)
	for _, char := range chars {
		// Better decentralized hash
		// s[0]*31^(n-1) + s[1]*31^(n-2) + ... + s[n-1]
		hash = 31*hash + int(char)
	}

	return hash
}
