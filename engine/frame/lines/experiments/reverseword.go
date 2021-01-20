package main

// https://rtlstyling.com/posts/rtl-styling/

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"os"
	"unicode"
)

func main() {
	args := os.Args[1:]
	if len(args) > 0 {
		html(rlText)
		return
	}
	scanner := bufio.NewScanner(os.Stdin)
	i := 0
	revsentence := ""
	for scanner.Scan() {
		i++
		//fmt.Println(scanner.Text())
		s := scanner.Text()
		rev := reverseSentence(s)
		fmt.Printf("// %3d:  '%s'\n", i, s)
		fmt.Printf("//     ⇒ '%s'\n", rev)
		//unirev := unicoded(rev)
		//fmt.Printf("// %3d:  %s\n", i, unirev)
		revsentence += rev
	}
	fmt.Printf("var rlText string = \"")
	fmt.Printf(gocode(revsentence))
	//fmt.Printf(revsentence)
	fmt.Printf("\"\n")
	if err := scanner.Err(); err != nil {
		log.Println(err)
	}
}

func reverseSentence(s string) string {
	runes := []rune(s)
	var word []rune
	var sentence string
	for _, r := range runes {
		if (unicode.IsSpace(r) || unicode.IsPunct(r)) && len(word) > 0 {
			rev := reverseWord(word)
			sentence += rev
			word = word[:0]
		}
		if unicode.IsSpace(r) || unicode.IsPunct(r) {
			sentence += string(r)
		} else {
			word = append(word, r)
		}
	}
	if len(word) > 0 {
		rev := reverseWord(word)
		sentence += rev
	}
	return sentence
}

func reverseWord(r []rune) string {
	if len(r) == 0 {
		return ""
	}
	var letter bytes.Buffer
	letter.WriteRune(r[0])
	letterlen := letter.Len()
	//fmt.Printf("letterlen=%d\n", letterlen)
	if letterlen > 1 {
		for i, j := 0, len(r)-1; i < len(r)/2; i, j = i+1, j-1 {
			r[i], r[j] = r[j], r[i]
		}
	}
	return string(r)
}

// ReverseRunes returns its argument string reversed rune-wise left to right.
func ReverseRunes(s string) string {
	r := []rune(s)
	for i, j := 0, len(r)-1; i < len(r)/2; i, j = i+1, j-1 {
		r[i], r[j] = r[j], r[i]
	}
	return string(r)
}

func unicoded(s string) string {
	runes := []rune(s)
	out := ""
	first := true
	for _, r := range runes {
		if first {
			first = false
		} else {
			out += " | "
		}
		out += fmt.Sprintf("%#U", r)
	}
	return out
}

func gocode(s string) string {
	runes := []rune(s)
	out := ""
	//first := true
	for _, r := range runes {
		// if first {
		// 	first = false
		// } else {
		// out += " | "
		// }
		out += fmt.Sprintf("&#x%x;", r)
	}
	return out
}

func html(rl string) {
	fmt.Printf(`
<html>
    <!-- Text between angle brackets is an HTML tag and is not displayed.
    The information between the BODY and /BODY tags is displayed.-->
    <head>
    <title>This is my title, displayed at the top of the window.</title>
    <link rel="stylesheet" href="./test.css">
    </head>
    <!-- The information between the BODY and /BODY tags is displayed.-->
<body>
    <h1>Hello, 世界</h1>
    <p>Be <b>bold</b> in stating your key points. Put them in a list: </p>
    <ul>
    <li>The first <span class="red">item</span> in your list</li>
    <li>The second item; <i>italicize</i> key words</li>
	</ul>
	<p class="rltext">`)
	fmt.Printf(rl)
	fmt.Printf(`</p>
    <p>Improve your image by including an image. </p>
    <p><img src="https://freeiconshop.com/wp-content/uploads/edd/image-solid.png"
        alt="A Great HTML Resource"></p>
    <p>Add a link to your favorite <a href="https://www.dummies.com/">Web site</a>.
    Break up your page with a horizontal rule or two. </p>
    <hr>
    <!-- And add a copyright notice.-->
    <p>&#169; Wiley Publishing, 2011</p>
</body>
</html>
`)
}
