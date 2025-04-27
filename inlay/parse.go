package inlay

import "bytes"

var (
	thinkStart = []byte("<think>")
	thinkEnd   = []byte("</think>")
	codeWrap   = []byte("```")
)

func ParseContent(resp []byte) []byte {
	if bytes.HasPrefix(resp, thinkStart) {
		thinkEndIndex := bytes.Index(resp, thinkEnd)
		if thinkEndIndex > 0 {
			resp = resp[thinkEndIndex+len(thinkEnd):]
		}
	}

	if bytes.HasPrefix(resp, codeWrap) {
		firstNewline := bytes.Index(resp, []byte("\n"))
		if firstNewline > 0 {
			resp = resp[firstNewline+1:]
			lastBacktick := bytes.LastIndex(resp, codeWrap)
			if lastBacktick > 0 {
				resp = resp[:lastBacktick]
			}
			resp = bytes.TrimSpace(resp)
		}
	}

	return resp
}
