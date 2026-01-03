package flow_control_statements

import (
	"fmt"
)

/*

 A defer statement defers the execution of a function until the surrounding function returns.

The deferred call's arguments are evaluated immediately, but the function call is not executed until the surrounding function returns.

*/

func standardDefer() {
	defer fmt.Println("world")

	fmt.Println("hello")
}

/*

Deferred function calls are pushed onto a stack. When a function returns, its deferred calls are executed in last-in-first-out order.

*/

func stackingDefer() {
	fmt.Println("counting")

	for i := 0; i < 10; i++ {
		defer fmt.Println(i)
	}

	fmt.Println("done")
}

func deferStatement() {
	fmt.Println("Defer statement:")
	standardDefer()

	fmt.Println("\nStacking defer statement:")
	stackingDefer()

	//fmt.Println("\nMore on defer statement:")
	//moreOnDefer()
}

// Printing the characters in the file

//func moreOnDefer() {
//	file, err := CopyFile("output", "input")
//	if err != nil {
//		fmt.Printf("Error: %v\n", err) // ← Print the error!
//		return
//	}
//
//	fmt.Println(file)
//}

// Insecure method : leaking buffer

//func CopyFile(dstName, srcName string) (written int64, err error) {
//	src, err := os.Open(srcName)
//
//	if err != nil {
//		return
//	}
//
//	dst, err := os.Create(dstName)
//	if err != nil {
//		return
//	}
//
//	written, err = io.Copy(dst, src)
//	err = dst.Close()
//	if err != nil {
//		return 0, err
//	}
//	err = src.Close()
//	if err != nil {
//		return 0, err
//	}
//	return
//}

// Correct code based on handling of opening and closing of files

//func CopyFile(dstName, srcName string) (written int64, err error) {
//	src, err := os.Open(srcName)
//	if err != nil {
//		return
//	}
//	defer func(src *os.File) {
//		err := src.Close()
//		if err != nil {
//
//		}
//	}(src)
//
//	dst, err := os.Create(dstName)
//	if err != nil {
//		return // src.Close() will still execute via defer
//	}
//	defer func(dst *os.File) {
//		err := dst.Close()
//		if err != nil {
//
//		}
//	}(dst)
//
//	written, err = io.Copy(dst, src)
//	return // Both files closed automatically
//}
