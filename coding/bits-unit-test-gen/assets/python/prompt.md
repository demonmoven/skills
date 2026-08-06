# Python Language-Specific Prompt

## Function Extraction Methods

| Tool/Method | Description                              |
| ----------- | ---------------------------------------- |
| `grep`      | Quickly locate `def` / `class` declarations |
| `ast` module | Precisely parse function and class structures |
| `inspect`   | Retrieve function signatures and source at runtime |
| `pydoc`     | View module/function documentation       |

## Filtering Rules (Python-Specific)

**Skip:**

- `if __name__ == "__main__"` entry blocks
- Auto-generated code files (Proto-generated files, `*_pb2.py`, `*_pb2_grpc.py`, etc.)
- Files containing `# Code generated` or `@generated` comments
- Simple property getter/setter with fewer than 3 lines
- Functions with coverage exceeding 90%
- Pure type definition files (containing only `TypedDict`, `dataclass` declarations with no business logic)

## Test File Naming Conventions

| Item       | Convention                                                   |
| ---------- | ------------------------------------------------------------ |
| Test file  | `test_*.py` or `*_test.py`                                   |
| Location   | Same directory as source file (follow existing conventions)  |
| Test class | `Test{ClassName}` (e.g., `UserService` → `TestUserService`)  |
| Test function | `test_<func>_<scenario>`                                  |

---

## Processing Granularity

The minimum execution unit in Python is the **source file (module)**. Process selected functions and methods from the same module together so imports, fixtures, mocks, and test file updates stay coherent.

For each selected module, track the module path, selected functions or methods, starting lines, and class names for methods. Use that information directly when generating tests and when reporting coverage, verification commands, failures, and defect mappings.

---

## Execution Scheduling

Python uses the source file as the basic unit for task dispatch. The scheduling strategy:

1. **Sequential by file**: Process selected source files one at a time.
2. **Writer generates per file**: Writer handles the selected functions in the module, generating one test file or appending to an existing test file.
3. **Fixer verifies per file**: After Writer completes, dispatch Fixer to run tests for the specific test file, triage failures, and fix allowed test issues.
4. **Report per file**: Record generated test files, verification commands, status, failures, and defect mappings in the final response.

### Verify-and-Fix Rounds

Fixer's verify-and-fix loop for each test file is limited to a maximum of **3 rounds**.

---

## Pre-Check (Must be completed before writing tests)

> **⚠️ Hard prerequisite**: Before generating any test code, this step's project learning **must** be completed. Different projects vary greatly in testing frameworks, fixture patterns, and mock strategies. Skipping this step almost certainly leads to repeated fixes later.

### 1. Environment Detection

1. **Detect project configuration**: Check for `pyproject.toml`, `setup.py`, `setup.cfg` to understand project structure
2. **Detect test framework**: Confirm pytest (default) vs unittest — check for `pytest.ini`, `pyproject.toml [tool.pytest]`, or `conftest.py`
3. **Detect package manager**: pip, poetry (`pyproject.toml [tool.poetry]`), or pipenv (`Pipfile`)
4. **Check test directory layout**: Tests in `tests/` directory, alongside source files, or `src/` layout?
5. **Detect Python version** (recommended): Check `pyproject.toml` or `.python-version` — affects available syntax features (walrus operator, match-case, type hints, etc.)

### 2. Learn Project Testing Patterns

Learn the style of existing tests in the target file's test directory:

1. **Scan existing test files** (required): Read 1-2 existing `test_*.py` or `*_test.py` files to learn:
   - **Test framework**: pytest (function-based) vs unittest (class-based)?
   - **Mock strategy**: `unittest.mock.patch` decorator, `pytest-mock` (mocker fixture), or manual dependency injection?
   - **Assertion style**: bare `assert` (pytest) vs `self.assertEqual` (unittest)?
   - **Fixture patterns**: `@pytest.fixture`, `conftest.py` shared fixtures, or setup/teardown methods?
   - **Naming conventions**: Actual naming patterns for test functions/classes
   - **Test organization**: Flat functions, `@pytest.mark.parametrize`, or `Test*` classes grouping?
2. **Discover conftest.py and shared fixtures** (recommended):
   ```bash
   find . -name "conftest.py" | head -10
   grep -rn "@pytest.fixture\|def setup\|def teardown" --include="test_*.py" --include="conftest.py" . | head -10
   ```
   - If shared fixtures exist in `conftest.py`, prefer reusing them
3. **Read project conventions** (recommended): Check `AGENTS.md`, `CLAUDE.md` under `PROJECT_ROOT`
   - Extract unit-test-related requirements

### 3. Context Analysis

For each target function, gather sufficient context information:

1. **Layer 1 (required)**: Read the target function source code, understand function signature, type hints, default parameters, and class dependencies (if it's a method)
2. **Layer 2 (recommended)**: Read imported modules to understand external dependency signatures (for deciding mock strategy)
3. **Layer 3 (as needed)**: When Layer 2 information is insufficient, read indirect dependencies, dataclass/TypedDict definitions, or configuration modules

---

## Python Unit Test Standards

### Test Function/Method Signature

- Test functions must start with `test_` for pytest to auto-discover them
- Test classes must start with `Test` and cannot have an `__init__` method
- Test function parameters can declare fixture names for pytest auto-injection
- Prefer functional test style; only use test classes when multiple tests share setup logic

### Test Isolation Principles

- Each test function must be independent, not depending on execution order or side effects of other tests
- Passing state between test functions via module-level variables is forbidden
- When shared setup logic is needed, use fixtures (`@pytest.fixture`) rather than module-level variables
- Mock / patch must be set up and cleaned within the test function scope; leaking to other test cases is not allowed
- Use `with` statements or decorator-style `patch` to ensure automatic restoration

### Assertion Standards

- Prefer pytest native `assert` statements (pytest automatically provides detailed failure messages)
- For floating-point comparison, use `pytest.approx(expected, abs=delta)` or `assert abs(actual - expected) < delta`
- For set containment, use `assert item in collection` or `assert set_a <= set_b`
- For exception assertions, use `with pytest.raises(ExceptionType) as exc_info:` + verify message
- Prefer precise value comparison (`assert result == expected`), but when verifying a complex object is non-None before asserting its attributes, `assert result is not None` is a reasonable prerequisite assertion
- Using `assertTrue` / `assertEqual` and other `unittest`-style assertions is forbidden (unless the project has existing conventions)

### Exception Testing Requirements

- Functions that may raise exceptions must cover exception paths
- Use `pytest.raises` to assert exception type and verify exception message:
  ```python
  with pytest.raises(ValueError, match="不能为空"):
      service.process(None)
  ```
- When precise exception message matching is needed, use `exc_info.value` to access the exception object:
  ```python
  with pytest.raises(ValueError) as exc_info:
      service.process(None)
  assert "不能为空" in str(exc_info.value)
  ```

### None/Empty Value Handling

- Optional parameters need to cover `None` input scenarios
- `str` parameters need to cover both empty string `""` and `None` scenarios
- `list`/`dict`/`set` parameters need to cover both `None` and empty collection `[]`/`{}`/`set()` scenarios
- `Optional[T]` return values need to cover `None` scenarios
- Numeric parameters need to cover `0`, negative numbers, boundary values (e.g., `sys.maxsize`, `float('inf')`) scenarios

### Type Safety

- For functions using type hints, tests should cover type mismatch edge cases (if the function doesn't perform type checking)
- For functions returning `TypedDict` / `dataclass`, assert field by field rather than comparing the whole object (unless `__eq__` is implemented)

---

## Verification Methods

### Run Tests

```bash
python -m pytest <test_file> -v
```

### Coverage Check

```bash
python -m pytest <test_file> -v --cov=<source_module> --cov-report=term-missing
```

---

## Special Fix Rules

**Failure Triage:**

> For complete defect determination rules, see the "Failure Triage Process" section in `references/test-fixer/AGENT.md`. Only Python-specific supplements are listed here.

**Python-Specific Defect Signals (must be judged in context; cannot be directly determined as defects):**
- `TypeError: 'NoneType' object is not subscriptable/iterable/callable` → May be missing None guard, **but only counts as a defect when None arises from the function's internal logic**; triggered by tests intentionally passing None parameters doesn't count
- `AttributeError: 'NoneType' object has no attribute 'xxx'` → May be a missing None check, **same as above, need to exclude cases where tests intentionally pass None**
- `IndexError: list index out of range` → May be missing boundary check, **but only counts as a defect when empty list comes from a normal business scenario**
- `KeyError` → May be missing dict key existence check, **need to confirm whether the key comes from normal business input**
- `ZeroDivisionError` → May be missing division-by-zero guard, **most likely a real defect**
- `RecursionError: maximum recursion depth exceeded` → May be a recursion termination condition defect, **most likely a real defect**
- Assertion failure where expected value matches the correct semantics of the function → Logic defect, **most likely a real defect**

- If `import` reports `ModuleNotFoundError`, check whether dependencies are missing or `sys.path` is configured incorrectly
- If `patch` target path is wrong causing mock not to take effect, ensure the patch target is the **consumer's** import path, not the definer's
- If fixture reports `fixture 'xxx' not found`, check whether the fixture is defined in the correct `conftest.py`
- If `AttributeError: __enter__` or `__exit__` occurs, check whether `patch` was incorrectly used in non-context-manager form
- If async function tests report `RuntimeWarning: coroutine was never awaited`, install `pytest-asyncio` and use the `@pytest.mark.asyncio` decorator

---

## Formatting

```bash
black <file>
```

Or `autopep8 -i <file>` (follow project configuration).

If the project uses `ruff`:

```bash
ruff format <file>
```

---

## Code Style

Style is determined by priority: user instructions > AGENTS.md > existing tests in the same directory > defaults below.

| Item            | Convention                                                     |
| --------------- | -------------------------------------------------------------- |
| Comment language | Chinese                                                       |
| Naming          | `test_<func>_<scenario>` or `Test<Class>` (English only)      |
| File            | `test_<module>.py`, same directory as source file (follow existing conventions) |
| Assertion       | `pytest` + built-in `assert` (or follow existing tests)        |
| Mock            | `unittest.mock.patch` / `pytest-mock` (or follow existing tests) |
| Test case organization | Parameterize with `@pytest.mark.parametrize`            |
| import order    | stdlib → third-party → project internal modules, separated by blank lines |

---

## Mock Usage

### Basic Structure (unittest.mock)

Use `patch` decorator or context manager to mock external dependencies:

```python
from unittest.mock import patch, MagicMock

def test_get_user_info_normal_fetch():
    # 1. setup mock
    # 2. call target function
    # 3. assert results
    pass
```

### patch Decorator Form

```python
@patch("module_under_test.query_user_from_db")
def test_get_user_info_normal_fetch(mock_query):
    mock_query.return_value = User(id=1001, name="张三")

    result = get_user_info(1001)

    assert result.name == "张三"
    assert result.id == 1001
    mock_query.assert_called_once_with(1001)
```

### patch Context Manager Form

```python
def test_get_user_info_db_query_failed():
    with patch("module_under_test.query_user_from_db") as mock_query:
        mock_query.side_effect = ConnectionError("connection refused")

        with pytest.raises(ServiceError, match="查询失败"):
            get_user_info(1001)

        mock_query.assert_called_once_with(1001)
```

### pytest-mock Form (mocker fixture)

```python
def test_get_user_info_normal_fetch(mocker):
    mock_query = mocker.patch("module_under_test.query_user_from_db")
    mock_query.return_value = User(id=1001, name="张三")

    result = get_user_info(1001)

    assert result.name == "张三"
    mock_query.assert_called_once_with(1001)
```

### Mock Class Methods

```python
def test_user_service_get_profile_normal_fetch(mocker):
    mock_dao = mocker.patch.object(UserDAO, "find_by_id")
    mock_dao.return_value = UserProfile(name="李四", email="lisi@example.com")

    svc = UserService()
    profile = svc.get_profile(1001)

    assert profile.name == "李四"
    assert profile.email == "lisi@example.com"
    mock_dao.assert_called_once_with(1001)
```

### Mock Anti-Patterns (Forbidden)

- ❌ Mock simple utility functions (e.g., `str.split`, `len`, `sorted`) → ✅ Call directly, no mock needed
- ❌ Mock private methods (`_internal_method`) via `patch.object` → ✅ Cover them indirectly through public methods
- ❌ Patch too broadly (e.g., entire module) making tests meaningless → ✅ Only mock uncontrollable external dependencies (DB/HTTP/filesystem)
- ❌ Use `MagicMock` without `spec` allowing any attribute access without error → ✅ Always use `spec=True` or `autospec=True`
- ❌ Mock the function under test itself → ✅ Mock its **dependencies**, not the target
- ❌ Forget to assert mock was called (mock hides real failures silently) → ✅ Always verify `assert_called_once_with` or check return values

---

## Examples

### Example 1: Parameterized Tests (Recommended Pattern)

Target function:

```python
def add(a: int, b: int) -> int:
    return a + b
```

Test code:

```python
import sys
import pytest

@pytest.mark.parametrize(
    "a, b, expected",
    [
        pytest.param(1, 2, 3, id="two_positive_numbers"),
        pytest.param(5, -3, 2, id="positive_plus_negative"),
        pytest.param(-1, -2, -3, id="two_negative_numbers"),
        pytest.param(10, 0, 10, id="add_zero"),
        pytest.param(0, 0, 0, id="both_zero"),
        pytest.param(sys.maxsize, 1, sys.maxsize + 1, id="large_number_addition"),
    ],
)
def test_add(a, b, expected):
    assert add(a, b) == expected
```

### Example 2: Test Class Organization (Multiple Scenarios for One Method)

Target method:

```python
class UserService:
    def __init__(self, user_dao: UserDAO):
        self._user_dao = user_dao

    def get_user(self, user_id: int) -> User:
        if user_id is None or user_id <= 0:
            raise ValueError("用户ID不合法")
        user = self._user_dao.find_by_id(user_id)
        if user is None:
            raise UserNotFoundError(f"用户不存在: {user_id}")
        return user
```

Test code:

```python
from unittest.mock import MagicMock
import pytest


class TestUserService:

    @pytest.fixture
    def user_dao(self):
        return MagicMock(spec=UserDAO)

    @pytest.fixture
    def svc(self, user_dao):
        return UserService(user_dao)

    def test_get_user_normal_fetch(self, svc, user_dao):
        # 正常获取用户
        expected = User(id=1001, name="张三")
        user_dao.find_by_id.return_value = expected

        result = svc.get_user(1001)

        assert result.name == "张三"
        user_dao.find_by_id.assert_called_once_with(1001)

    def test_get_user_not_found_raises_error(self, svc, user_dao):
        # 用户不存在时抛出异常
        user_dao.find_by_id.return_value = None

        with pytest.raises(UserNotFoundError, match="用户不存在"):
            svc.get_user(999)

    def test_get_user_none_id_raises_value_error(self, svc):
        # ID为None时抛出参数异常
        with pytest.raises(ValueError, match="用户ID不合法"):
            svc.get_user(None)

    def test_get_user_negative_id_raises_value_error(self, svc):
        # ID为负数时抛出参数异常
        with pytest.raises(ValueError, match="用户ID不合法"):
            svc.get_user(-1)
```

---

## Common Pitfalls and Fixes

| Pitfall                                               | Cause                                                     | Fix                                                                 |
| ----------------------------------------------------- | --------------------------------------------------------- | ------------------------------------------------------------------- |
| `patch` not taking effect, target function still calls real implementation | Patch path is not the consumer's import path               | Patch target must be the **reference in the consumer module**, e.g., `module_a.func` not `module_b.func` |
| `fixture 'xxx' not found`                             | Fixture not defined in correct `conftest.py` or not imported | Place fixture in `conftest.py` at the same or parent directory level of the test file |
| `assert` statements optimized away by Python          | Runtime used `-O` optimization flag                        | Don't use `-O` when running tests; pytest doesn't optimize by default |
| `MagicMock` returns new Mock for any attribute access without error | MagicMock's default behavior is chain-generating child Mocks | Use `spec=True` or `autospec=True` to restrict attribute access     |
| Counter-intuitive parameter order when stacking `patch` decorators | Multiple `@patch` execute bottom-to-top, parameters inject left-to-right | Note decorator order: the innermost `@patch` corresponds to the first parameter |
| Async tests not executing                             | Missing `pytest-asyncio` or `@pytest.mark.asyncio` not added | Install `pytest-asyncio` and add the decorator                      |
| Object parameters in `parametrize` cause unreadable test IDs | Default shows object's `repr`                              | Use `pytest.param(..., id="description")` to specify readable IDs   |
| Tests affecting each other                            | Module-level variables or singletons modified without restoration | Use `yield` + cleanup logic in fixtures, or use `monkeypatch`       |

---

## Context Discovery Commands

```bash
grep -rn "def \|class " <file>
pip list | grep -i test
cat setup.py setup.cfg pyproject.toml 2>/dev/null
find . -name "test_*.py" -o -name "*_test.py" | head -20
grep -rn "import.*mock\|from.*mock\|@patch\|@pytest" --include="test_*.py" . | head -10
grep -rn "conftest\|fixture" --include="conftest.py" .
```
