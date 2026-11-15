// Command fakegh stands in for the `gh` binary in cmd's tests: it logs each
// call and answers from the first matching rule in $GH_FAKE_RULES. It is a
// separate tiny program so each fake call costs milliseconds, not a
// re-execution of the whole test binary.
package main

import (
	"encoding/json"
	"os"
	"strings"
)

type rule struct {
	Match  string
	Stdout string
	Fail   string
}

func main() {
	line, _ := json.Marshal(os.Args[1:])
	if f, err := os.OpenFile(os.Getenv("GH_FAKE_LOG"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644); err == nil {
		f.Write(append(line, '\n'))
		f.Close()
	}

	var rules []rule
	data, _ := os.ReadFile(os.Getenv("GH_FAKE_RULES"))
	json.Unmarshal(data, &rules)
	joined := strings.Join(os.Args[1:], " ")
	for _, r := range rules {
		if strings.Contains(joined, r.Match) {
			if r.Fail != "" {
				os.Stderr.WriteString(r.Fail)
				os.Exit(1)
			}
			os.Stdout.WriteString(r.Stdout)
			return
		}
	}
}
