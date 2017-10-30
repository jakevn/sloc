sloc is a command-line program for quickly counting lines of code recursively within the current directory.
Simplicity was a major design goal, and as a result, the implementation is under 250 lines of Go utilizing only the 
standard library. Performance was a secondary goal, and parallelism is employed.

Supported Languages
---
* Python
* Go
* Ruby
* C
* C++
* C#
* JavaScript
* JSON
* Java
* YAML
* HTML
* CSS
* Shell
* Powershell
* Rust

Example Output and Performance
---
Example when run on modern hardware (i7 quad-core laptop with SSD):

```
jakevn$ time sloc
YAML
  Lines: 91
  Chars: 2,484
  Files: 2
  Chars/Line: 27
  Lines/File: 45
Shell
  Lines: 107
  Chars: 4,148
  Files: 6
  Chars/Line: 38
  Lines/File: 17
Java
  Lines: 10,297
  Chars: 468,077
  Files: 62
  Chars/Line: 45
  Lines/File: 166
JavaScript
  Lines: 274
  Chars: 8,914
  Files: 3
  Chars/Line: 32
  Lines/File: 91
C
  Lines: 929,458
  Chars: 41,148,832
  Files: 3,203
  Chars/Line: 44
  Lines/File: 290
HTML
  Lines: 336
  Chars: 9,942
  Files: 1
  Chars/Line: 29
  Lines/File: 336
Python
  Lines: 4,906
  Chars: 206,655
  Files: 56
  Chars/Line: 42
  Lines/File: 87
JSON
  Lines: 3,272
  Chars: 115,231
  Files: 1
  Chars/Line: 35
  Lines/File: 3,272
C#
  Lines: 3,539
  Chars: 153,138
  Files: 31
  Chars/Line: 43
  Lines/File: 114
C++
  Lines: 310,714
  Chars: 13,827,423
  Files: 791
  Chars/Line: 44
  Lines/File: 392
Total Lines: 1,262,994
Total Chars: 55,944,844
Total Files: 4,156

real	0m0.623s
user	0m4.292s
sys	0m0.203s
```

Limitations
---
Multi-line (block) comments are not currently supported. This is largely due to the focus on simplicity and speed.


Adding A Language
---
It is simple to add support for a language. Near the top of `main.go`, you will find a map of language definitions.

For example:

```
var languages = map[string]lang{
	"Python": {
		Filetypes:       []string{".py"},
		Comment:         []string{"#"},
		IgnoreIfOnly:    commonIgnore,
		TestFilename:    []string{"test_"},
		TestDirname:     []string{"test"},
		DirectoryIgnore: []string{"venv", "egg-info"},
	},
	"Go": {
		Filetypes:       []string{".go"},
		Comment:         []string{"//"},
		IgnoreIfOnly:    commonIgnore,
		DirectoryIgnore: []string{"vendor"},
		TestFilename:    []string{"_test"},
	},

... etc
```

You can add a language by adding another `lang` struct to this map.