import random


class CrawlPacing:
    def __init__(self, minimum_seconds: float = 5.0, maximum_seconds: float = 30.0):
        self.minimum_seconds = minimum_seconds
        self.maximum_seconds = maximum_seconds

    def delay_seconds(self, configured_delay_seconds: float) -> float:
        return random.uniform(
            max(self.minimum_seconds, configured_delay_seconds), self.maximum_seconds
        )
