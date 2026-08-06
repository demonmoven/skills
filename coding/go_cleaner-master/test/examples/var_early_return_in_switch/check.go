// RUN: %cleaner -f --passes=clean-after-return,clean-unused-decl
// RUN: cat %s | %filecheck %s
package main

import (
	"context"
	"fmt"
	"sync"
)

func mockFun(ctx context.Context, id int64) (int64, error) {
	return 1, nil
}

func run(ctx context.Context) error {
	constant, err := mockFun(ctx, 2)
	if err != nil {
		return fmt.Errorf("error")
	}
	switch constant {
	case 1:
		return nil
		_, err := mockFun(ctx, 3)
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

	case 2:
		_, err := mockFun(ctx, 5)
		if err != nil {
			return err
		}
	case 3:
		_, err := mockFun(ctx, 5)
		return nil
		// this comment should be removed
		// this comment should be removed
		if err != nil {
			return err
		}
	}

	return nil
}

func main() {
	run(context.Background())
}

// CHECK: switch constant {
// CHECK-NEXT: case 1:
// CHECK-NEXT: 	return nil
// CHECK-EMPTY:
// CHECK-NEXT: case 2:
// CHECK-NEXT: 	_, err := mockFun(ctx, 5)
// CHECK-NEXT: 	if err != nil {
// CHECK-NEXT: 		return err
// CHECK-NEXT: 	}
// CHECK-NEXT: case 3:
// CHECK-NEXT: 	_, _ = mockFun(ctx, 5)
// CHECK-NEXT: 	return nil
// CHECK-EMPTY:
// CHECK-NEXT: 	}
