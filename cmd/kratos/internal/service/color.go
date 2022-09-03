package service

import (
	"fmt"
	"os"
	"strconv"
)

func colorString(l Level, s string) string {
	switch l {
	case LevelDebug:
		return Colorize(s, FgHiMagenta)
	case LevelInfo:
		return Colorize(s, FgHiGreen)
	case LevelWarn:
		return Colorize(s, FgHiYellow)
	case LevelError:
		return Colorize(s, FgHiRed)
	case LevelFatal:
		return Colorize(s, FgHiRed)
	}
	return s
}

type Level int8

const (
	// LevelDebug is logger debug level.
	LevelDebug Level = iota - 1
	// LevelInfo is logger info level.
	LevelInfo
	// LevelWarn is logger warn level.
	LevelWarn
	// LevelError is logger error level.
	LevelError
	// LevelFatal is logger fatal level
	LevelFatal
)

//第一行是红字黑底，第二行红底白字。
//
//我们来解析 \033[1;31;40m%s\033[0m\n 这个字符串中的字符分别代表了什么。
//
//\033：\ 表示转义，\033 表示设置颜色。
//[1;31;40m：定义颜色，[ 表示开始颜色设置，m 为颜色设置结束，以 ; 号分隔。1 代码，表示显示方式，31 表示前景颜色（文字的 颜色），40 表示背景颜色。
//\033[0m：表示恢复终端默认样式

// 前景 背景 颜色
// ---------------------------------------
// 30  40  黑色
// 31  41  红色
// 32  42  绿色
// 33  43  黄色
// 34  44  蓝色
// 35  45  紫红色
// 36  46  青蓝色
// 37  47  白色

// 3 位前景色, 4 位背景色

// 代码 意义
// -------------------------
//  0  终端默认设置
//  1  高亮显示
//  4  使用下划线
//  5  闪烁
//  7  反白显示
//  8  不可见

// Color defines a single SGR Code
type Color int

// Foreground text colors
const (
	FgBlack Color = iota + 30
	FgRed
	FgGreen
	FgYellow
	FgBlue
	FgMagenta
	FgCyan
	FgWhite
)

// Foreground Hi-Intensity text colors
const (
	FgHiBlack Color = iota + 90
	FgHiRed
	FgHiGreen
	FgHiYellow
	FgHiBlue
	FgHiMagenta
	FgHiCyan
	FgHiWhite
)

// Colorize a string based on given color.
func Colorize(s string, c Color) string {
	return fmt.Sprintf("\033[1;%s;40m%s\033[0m", strconv.Itoa(int(c)), s)
}
func PinkLog(format string, a ...interface{}) {
	fmt.Fprintf(os.Stdout, fmt.Sprintf("\033[1;%s;40m%s\033[0m\n", strconv.Itoa(int(FgHiMagenta)), format), a)
}
func GreenLog(format string, a ...interface{}) {
	fmt.Fprintf(os.Stdout, fmt.Sprintf("\033[1;%s;40m%s\033[0m\n", strconv.Itoa(int(FgHiGreen)), format), a)
}
func RedLog(format string, a ...interface{}) {
	fmt.Fprintf(os.Stdout, fmt.Sprintf("\033[1;%s;40m%s\033[0m\n", strconv.Itoa(int(FgHiRed)), format), a)
}
func YellowLog(format string, a ...interface{}) {
	fmt.Fprintf(os.Stdout, fmt.Sprintf("\033[1;%s;40m%s\033[0m\n", strconv.Itoa(int(FgHiYellow)), format), a)
}
func FatalLog(format string, a ...interface{}) {
	fmt.Fprintf(os.Stderr, fmt.Sprintf("\033[1;%s;40m%s\033[0m\n", strconv.Itoa(int(FgRed)), format), a)
	os.Exit(-1)
}
