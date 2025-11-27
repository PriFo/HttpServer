from __future__ import annotations

import threading
from concurrent.futures import Future
from queue import PriorityQueue, Empty
from typing import Any, Callable


class PriorityScheduler:
    """Simple priority-aware task executor for serialized model access."""

    def __init__(self, max_workers: int = 4):
        self.max_workers = max_workers
        self._queue: PriorityQueue = PriorityQueue()
        self._threads: list[threading.Thread] = []
        self._task_counter = 0
        self._lock = threading.Lock()
        self._stopping = threading.Event()
        self._active_workers = 0
        self._completed_tasks = 0
        self._error_tasks = 0
        self._start_workers()

    def _start_workers(self) -> None:
        for idx in range(self.max_workers):
            thread = threading.Thread(target=self._worker, name=f"priority-worker-{idx}", daemon=True)
            thread.start()
            self._threads.append(thread)

    def _worker(self) -> None:
        while not self._stopping.is_set():
            try:
                _, _, func, args, kwargs, future = self._queue.get(timeout=0.5)
            except Empty:
                continue
            with self._lock:
                self._active_workers += 1
            if future.set_running_or_notify_cancel():
                try:
                    result = func(*args, **kwargs)
                    future.set_result(result)
                    with self._lock:
                        self._completed_tasks += 1
                except Exception as exc:
                    future.set_exception(exc)
                    with self._lock:
                        self._error_tasks += 1
            self._queue.task_done()
            with self._lock:
                self._active_workers = max(0, self._active_workers - 1)

    def submit(
        self,
        priority: float,
        func: Callable[..., Any],
        *args: Any,
        **kwargs: Any,
    ) -> Future:
        with self._lock:
            count = self._task_counter
            self._task_counter += 1
        future: Future = Future()
        # negate priority because PriorityQueue pops smallest first
        self._queue.put((-priority, count, func, args, kwargs, future))
        return future

    def shutdown(self, wait: bool = False) -> None:
        self._stopping.set()
        if wait:
            for thread in self._threads:
                thread.join(timeout=1.0)

    def snapshot(self) -> dict:
        with self._lock:
            return {
                "active_workers": self._active_workers,
                "queued_tasks": self._queue.qsize(),
                "completed_tasks": self._completed_tasks,
                "error_tasks": self._error_tasks,
            }

