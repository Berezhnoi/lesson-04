# Аналіз дедлоку (Частина 2)

## Код, що аналізується

```go
package main

import (
	"fmt"
	"sync"
)

func main() {
	var wg sync.WaitGroup
	ch := make(chan int) // небуферизований канал

	wg.Add(1)
	go func() {
		// симуляція важкої роботи
		ch <- 42
		wg.Done()
	}()

	wg.Wait()
	fmt.Println(<-ch)
}
```

## Мій промпт до ШІ

Вставте нижче точний текст промпту, який ви розробили. Промпт має
явно вимагати від ШІ:

- **не просто переписати код**, а пояснити точну послідовність
  переплетення (interleaving) горутин, що веде до зависання;
- проаналізувати, чому виклик `wg.Wait()` перед читанням з
  небуферизованого каналу зупиняє виконання.

### Мій промпт

Analyze this Go code and explain exactly why it deadlocks.

Do not rewrite or fix it. I need to understand the execution flow.

Explain:
1. the sequence of events between the main goroutine and the spawned goroutine;
2. what happens when the goroutine reaches `ch <- 42`;
3. why the unbuffered channel blocks the sender;
4. what the main goroutine is doing at `wg.Wait()`;
5. why `wg.Done()` can never be reached;
6. why `<-ch` is never executed;
7. the circular waiting/dependency that causes the deadlock.

Focus on the original code and its execution order. Do not just say “both goroutines are waiting.” Explain exactly what each goroutine is waiting for. Do not assume the channel can store the value `42` because it is unbuffered. Use simple Go terminology and a step-by-step timeline. At the end, briefly describe what ordering needs to change to avoid the deadlock, but do not provide a complete rewritten solution unless necessary.

## Пояснення від ШІ

Вставте нижче відповідь ШІ-асистента на ваш промпт — покрокове
пояснення переплетення горутин.

1. The main goroutine starts the goroutine, then immediately calls `wg.Wait()`.
2. The spawned goroutine reaches `ch <- 42` first. But `ch` is unbuffered, so this send cannot complete unless another goroutine is already ready to receive.
3. The sender blocks at `ch <- 42` and cannot continue to `wg.Done()`.
4. The main goroutine is blocked at `wg.Wait()` waiting for the counter to reach zero.
5. That counter will only reach zero after `wg.Done()` runs, but `wg.Done()` is never reached because the send is blocked.
6. Since the main goroutine is still waiting, it never reaches `fmt.Println(<-ch)`.
7. The deadlock is circular: the goroutine waits for the main goroutine to receive from the channel, while the main goroutine waits for the goroutine to call `wg.Done()`. Each side waits for the other to do the first step.

The key issue is ordering: the receive must happen before the sender blocks, or the sender must not wait on a receive before the main goroutine is free to receive.

### Відповідь ШІ

## Мінімальне виправлення

Робочу, тестовану версію виправлення реалізовано в
[`deadlock/fixed.go`](./deadlock/fixed.go) (функція `Run`), а
автоматична перевірка — у `deadlock/fixed_test.go`. Я змінив порядок так, що
горутина просто надсилає `42` у буферизований канал, а потім основна горутина
зчитує значення й повертає його. Це усуває циклічне очікування, бо відправник
більше не блокується на `Send` в очікуванні на одночасний `Receive`.
