import os
import sys
import tempfile
import unittest


# Ensure we can import validate_pipeline.py from this directory.
sys.path.append(os.path.dirname(os.path.abspath(__file__)))

from validate_pipeline import _load_payload, validate_pipeline


def _valid_job(job_id: str = "job-1", **overrides):
    job = {
        "id": job_id,
        "name": "Job 1",
        "uses": "some-atom",
        "if": "true",
        "depends_on": [],
    }
    job.update(overrides)
    return job


def _valid_stage(stage_id: str = "stage-1", job_id: str = "job-1", **overrides):
    stage = {
        "id": stage_id,
        "name": {"value": "Stage 1"},
        "if": "true",
        "jobs": [_valid_job(job_id)],
    }
    stage.update(overrides)
    return stage


def _valid_payload(**overrides):
    payload = {
        "name": {"value": "pipeline"},
        "stages": [_valid_stage()],
    }
    payload.update(overrides)
    return payload


class TestValidatePipeline(unittest.TestCase):
    def test_valid_payload(self):
        errors = validate_pipeline(_valid_payload())
        self.assertEqual(errors, [], f"valid payload should pass, got: {errors}")

    def test_invalid_json_string(self):
        payload = _load_payload("{invalid-json", "")
        self.assertIn("_error", payload)
        errors = validate_pipeline(payload)
        self.assertTrue(errors and errors[0].startswith("Invalid JSON string"))

    def test_load_payload_file_error(self):
        payload = _load_payload("", "this_file_should_not_exist_123456.json")
        self.assertIn("_error", payload)
        self.assertTrue(payload["_error"].startswith("Failed to read file"))

    def test_load_payload_file_ok(self):
        with tempfile.NamedTemporaryFile(mode="w", encoding="utf-8", suffix=".json", delete=True) as f:
            f.write('{"name": {"value": "p"}, "stages": []}')
            f.flush()
            payload = _load_payload("", f.name)
        self.assertIsInstance(payload, dict)
        self.assertEqual(payload.get("name", {}).get("value"), "p")

    def test_missing_root_required_fields(self):
        errors = validate_pipeline({})
        self.assertTrue(any("'name' is a required property" in e for e in errors), errors)
        self.assertTrue(any("'stages' is a required property" in e for e in errors), errors)

    def test_missing_root_name_value(self):
        errors = validate_pipeline({"name": {}, "stages": []})
        self.assertTrue(any("'value' is a required property" in e for e in errors), errors)

    def test_stage_required_fields(self):
        payload = _valid_payload(stages=[{"id": "s"}])
        errors = validate_pipeline(payload)
        self.assertTrue(any("'name' is a required property" in e for e in errors), errors)
        self.assertTrue(any("'jobs' is a required property" in e for e in errors), errors)
        self.assertTrue(any("'if' is a required property" in e for e in errors), errors)

    def test_stage_additional_properties_forbidden(self):
        payload = _valid_payload(stages=[_valid_stage(extra_field=1)])
        errors = validate_pipeline(payload)
        self.assertTrue(any("Additional properties are not allowed" in e for e in errors), errors)

    def test_jobs_min_items(self):
        payload = _valid_payload(stages=[_valid_stage(jobs=[])])
        errors = validate_pipeline(payload)
        self.assertTrue(
            any("should be non-empty" in e or "is too short" in e for e in errors),
            errors,
        )

    def test_job_required_fields(self):
        payload = _valid_payload(stages=[_valid_stage(jobs=[{"id": "j"}])])
        errors = validate_pipeline(payload)
        self.assertTrue(any("'name' is a required property" in e for e in errors), errors)
        self.assertTrue(any("'uses' is a required property" in e for e in errors), errors)
        self.assertTrue(any("'if' is a required property" in e for e in errors), errors)
        self.assertTrue(any("'depends_on' is a required property" in e for e in errors), errors)

    def test_job_additional_properties_forbidden(self):
        payload = _valid_payload(stages=[_valid_stage(jobs=[_valid_job(extra_field=1)])])
        errors = validate_pipeline(payload)
        self.assertTrue(any("Additional properties are not allowed" in e for e in errors), errors)

    def test_job_id_pattern(self):
        payload = _valid_payload(stages=[_valid_stage(jobs=[_valid_job(job_id="1invalid")])])
        errors = validate_pipeline(payload)
        self.assertTrue(any("does not match" in e for e in errors), errors)

    def test_depends_on_items_type(self):
        payload = _valid_payload(stages=[_valid_stage(jobs=[_valid_job(depends_on=[1])])])
        errors = validate_pipeline(payload)
        self.assertTrue(any("is not of type 'string'" in e for e in errors), errors)

    def test_timeout_range(self):
        payload = _valid_payload(stages=[_valid_stage(jobs=[_valid_job(timeout=-1)])])
        errors = validate_pipeline(payload)
        self.assertTrue(any("less than the minimum" in e for e in errors), errors)

        payload = _valid_payload(stages=[_valid_stage(jobs=[_valid_job(timeout=7776001)])])
        errors = validate_pipeline(payload)
        self.assertTrue(any("greater than the maximum" in e for e in errors), errors)

    def test_retry_enum(self):
        payload = _valid_payload(stages=[_valid_stage(jobs=[_valid_job(retry={"max": 10})])])
        errors = validate_pipeline(payload)
        self.assertTrue(any("is not one of" in e for e in errors), errors)

    def test_notification_type_enum(self):
        payload = _valid_payload(
            stages=[
                _valid_stage(
                    jobs=[
                        _valid_job(
                            notifications=[
                                {"type": "unknown"},
                            ]
                        )
                    ]
                )
            ]
        )
        errors = validate_pipeline(payload)
        self.assertTrue(any("is not one of" in e for e in errors), errors)

    def test_stage_max_items(self):
        stages = []
        for i in range(17):
            stages.append(_valid_stage(stage_id=f"stage-{i}", job_id=f"job-{i}"))
        payload = _valid_payload(stages=stages)
        errors = validate_pipeline(payload)
        self.assertTrue(any("is too long" in e for e in errors), errors)

    def test_business_duplicate_stage_id(self):
        payload = _valid_payload(
            stages=[
                _valid_stage(stage_id="dup", job_id="job-1"),
                _valid_stage(stage_id="dup", job_id="job-2"),
            ]
        )
        errors = validate_pipeline(payload)
        self.assertIn("stages[1].id: duplicate stage id 'dup'", errors)

    def test_business_duplicate_job_id_global(self):
        payload = _valid_payload(
            stages=[
                _valid_stage(stage_id="s1", job_id="job-x"),
                _valid_stage(stage_id="s2", job_id="job-x"),
            ]
        )
        errors = validate_pipeline(payload)
        self.assertIn("duplicate job id job-x", errors)

    def test_business_job_self_dependency(self):
        payload = _valid_payload(
            stages=[
                _valid_stage(
                    stage_id="s1",
                    jobs=[_valid_job(job_id="job-x", depends_on=["job-x"])],
                )
            ]
        )
        errors = validate_pipeline(payload)
        self.assertIn("Job 'job-x' cannot depend on itself", errors)


if __name__ == "__main__":
    unittest.main()
