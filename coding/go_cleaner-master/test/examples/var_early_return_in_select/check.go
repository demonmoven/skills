// RUN: %cleaner -f --passes=clean-after-return,clean-unused-decl
// RUN: cat %s | %filecheck %s
package main

import (
	"context"
	"fmt"
	"sync"
)

func mockFun(ctx context.Context, id int64) (<-chan int64, error) {
	ch := make(chan int64)
	return ch, nil
}

func run(ctx context.Context) error {
	ch1, err := mockFun(ctx, 1)
	ch2, err := mockFun(ctx, 2)
	ch3, err := mockFun(ctx, 3)
	if err != nil {
		return fmt.Errorf("error")
	}
	select {
	case msg1 := <-ch1:
		return nil
		_, err := mockFun(ctx, msg1)
		if err != nil {
			return err
		}

		group := sync.WaitGroup{}
		group.Add(1)
		go mockFun(ctx, 4)

		workers := 5
		for i := 0; i < workers; i++ {
			group.Add(1)
			go func() {
				defer group.Done()
			}()
		}

		group.Wait()

	case _ = <-ch2:
		_, err := mockFun(ctx, 5)
		if err != nil {
			return err
		}
	case msg3, ok := <-ch3:
		_, err := mockFun(ctx, msg3)
		return nil
		// this comment should be removed
		// this comment should be removed
		if err != nil && ok {
			return fmt.Errorf("error%d", msg3)
		}
	}

	return nil
}

func main() {
	run(context.Background())
}

// CHECK: select {
// CHECK-NEXT: case _ = <-ch1:
// CHECK-NEXT:         return nil
// CHECK-EMPTY:
// CHECK-NEXT: case _ = <-ch2:
// CHECK-NEXT:         _, err := mockFun(ctx, 5)
// CHECK-NEXT:         if err != nil {
// CHECK-NEXT:                 return err
// CHECK-NEXT:         }
// CHECK-NEXT: case msg3, _ := <-ch3:
// CHECK-NEXT:         _, _ = mockFun(ctx, msg3)
// CHECK-NEXT:         return nil
// CHECK-EMPTY:
// CHECK-NEXT: }
