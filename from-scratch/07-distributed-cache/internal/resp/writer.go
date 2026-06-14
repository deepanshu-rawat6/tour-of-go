package resp

import "fmt"

func SimpleString(s string) string  { return fmt.Sprintf("+%s\r\n", s) }
func Error(msg string) string       { return fmt.Sprintf("-ERR %s\r\n", msg) }
func Integer(n int64) string        { return fmt.Sprintf(":%d\r\n", n) }
func NullBulk() string              { return "$-1\r\n" }
func BulkString(s string) string    { return fmt.Sprintf("$%d\r\n%s\r\n", len(s), s) }
func Array(items []string) string {
	out := fmt.Sprintf("*%d\r\n", len(items))
	for _, item := range items {
		out += BulkString(item)
	}
	return out
}
