from src.adapters.redis.standard import RedisAdapter


class FakeRedisClient:
    def __init__(self):
        self.sets = {}

    def sadd(self, key, *values):
        members = self.sets.setdefault(key, set())
        before = len(members)
        members.update(values)
        return len(members) - before

    def smembers(self, key):
        return self.sets.get(key, set())

    def srem(self, key, *values):
        members = self.sets.setdefault(key, set())
        before = len(members)
        for value in values:
            members.discard(value)
        return before - len(members)

    def sinter(self, *keys):
        if not keys:
            return set()
        sets = [self.sets.get(key, set()) for key in keys]
        return set.intersection(*sets)


def adapter_with_fake_client():
    adapter = RedisAdapter.__new__(RedisAdapter)
    adapter.client = FakeRedisClient()
    return adapter


def test_standard_redis_adapter_set_members_round_trip_strings():
    adapter = adapter_with_fake_client()

    assert adapter.sadd("nodes:index", "node_a:v1.0.0", "node_b:v1.0.0") == 2

    assert adapter.smembers("nodes:index") == {"node_a:v1.0.0", "node_b:v1.0.0"}


def test_standard_redis_adapter_srem_serializes_values_before_removal():
    adapter = adapter_with_fake_client()
    adapter.sadd("nodes:index", "node_a:v1.0.0", "node_b:v1.0.0")

    assert adapter.srem("nodes:index", "node_a:v1.0.0") == 1

    assert adapter.smembers("nodes:index") == {"node_b:v1.0.0"}


def test_standard_redis_adapter_sinter_deserializes_intersection():
    adapter = adapter_with_fake_client()
    adapter.sadd("nodes:tags:test", "node_a:v1.0.0", "node_b:v1.0.0")
    adapter.sadd("nodes:tags:agent", "node_b:v1.0.0")

    assert adapter.sinter("nodes:tags:test", "nodes:tags:agent") == {"node_b:v1.0.0"}
