import os
import json
import queue
import subprocess
import threading
import concurrent.futures
from .testcase import TestCase, TestResult

WorkerLock = threading.Lock()
SubProcSet = set()

def doc_for_compatible(version):
    return f'Testing compatibility - Go {version}'

def doc_for_functional(version, fn):
    return f'Testing {fn} - Go {version}'

def doc_for_multiple_functional(version, *fns):
    return 'Multi-test Goals - Go {}:\n\n{}'.format(version, "\n".join(['  * '+ f for f in fns]) + '\n')

def new_test_case_from_json(name, test):
    doc = 'unknown goals'
    version = test['go_version']
    goals = test['goals']

    if len(goals) == 1:
        if goals[0] == 'compatibility':
            doc = doc_for_compatible(version)
        else:
            doc = doc_for_functional(version, goals[0])

    if len(goals) > 1:
        doc = doc_for_multiple_functional(version, *goals)
    
    return TestCase(name, test['head'], version, doc, test['build_flags'], test['cleaner_flags'])

def exec_docker_command(version, command, debug=False):
    version = '1.18' if version <= '1.18' else version
    command = f"docker exec -it worker_v2.1_go{version}_2_v0.0.1_container_01 /bin/bash -c '{command}'"
    if not debug:
        print('\033[32m' + command + '\033[0m')
        return subprocess.run(command, shell=True)
    else:
        print(command)
        return None

def cleaner_install_command(version):
    branch = 'master'
    command = 'go install code.byted.org/analyzers/go_cleaner/cmd/cleaner@{0}'
    if version >= '1.23':
        return command.format('release-branch.go1.23')

    proc = subprocess.run('git rev-parse --abbrev-ref HEAD', shell=True, capture_output=True, text=True)
    if proc.returncode:
        return command.format(branch)
    return command.format(proc.stdout.strip())


def exec_docker_command_parallel(container, version, command, debug=False):
    version = '1.18' if version <= '1.18' else version
    command = f"docker exec worker_v2.1_go{version}_{container}_v0.0.1_container_01 /bin/bash -c '{command}'"
    if not debug:
        return subprocess.run(command, shell=True, capture_output=True, text=True, preexec_fn=os.setsid), '\033[32m' + command + '\033[0m'
    else:
        return None, command

class BenchTask(object):
    def __init__(self, container, test_case, version, extra_flags, debug):
        self.container = container
        self.test_case = test_case
        self.version = version
        self.extra_flags = extra_flags
        self.debug = debug
    

def bench_worker(task):
    test_case = task.test_case
    version = task.version
    container = task.container
    extra_flags = task.extra_flags
    debug = task.debug

    result = TestResult(test_case)
    proc, msg = exec_docker_command_parallel(container, version, test_case.get_cleaner_command(extra_flags), debug)

    with WorkerLock:
        SubProcSet.add(proc)

    result.add_message(msg)
    result.set_proc(proc)
    if result.has_error():
        return result

    # Clean up
    command = f'cd /tmp && rm -rf {test_case.get_project()}'
    result.add_message(command)

    proc, msg = exec_docker_command_parallel(container, version, command, debug)
    
    with WorkerLock:
        SubProcSet.add(proc)
    
    result.add_message(msg)
    result.set_proc(proc)

    return result


class Benchmark:
    def __init__(self, test_file_path, jobs=24):
        self.jobs = jobs
        with open(test_file_path, 'r') as fp:
            tests = json.load(fp)
        
        self.cases = []
        self.targets = []

        for name, test in tests.items():
            self.cases.append(new_test_case_from_json(name, test))

    def set_target_cases(self, names: list):
        self.targets.extend(case for name in names for case in self.cases if case.repo == name)

    def is_target_case(self, target):
        return len(self.targets) == 0 or target in self.targets

    def get_repos_version(self, version):
        cases = [c for c in self.cases if self.is_target_case(c) and c.version == version]
        if not cases:
            print(f"Can't find cases for version {version}")

        return cases
    
    def get_all_version(self):
        versions = list(set([c.version for c in self.cases if self.is_target_case(c)]))
        return sorted(versions)

    def run_bench(self, versions, debug, extra_flags):
        for v in versions:
            command = cleaner_install_command(v)
            exec_docker_command(v, command, debug)

        total = sum([len(self.get_repos_version(v)) for v in versions])
        index = 1
        for v in versions:
            for case in self.get_repos_version(v):
                print(case.report_start(index, total))
                result = exec_docker_command(v, case.get_cleaner_command(extra_flags), debug)

                # Clean up
                command = f'cd /tmp && rm -rf {case.get_project()}'
                exec_docker_command(v, command, debug)

                # 出现错误即停止
                if result and result.returncode:
                    print(case.report_error(index, total))
                    index += 1
                    return result.returncode
                
                index += 1

        return 0

    def run_bench_with_tasks(self, tasks, jobs, finished, total, retry=True):
        failed_tasks = []
        taskmap = {}
        for t in tasks:
            taskmap[t.test_case.repo] = t

        with concurrent.futures.ThreadPoolExecutor(max_workers=jobs) as executor:
            futures = [executor.submit(bench_worker, t) for t in tasks]
            for future in concurrent.futures.as_completed(futures):
                try:
                    result = future.result()
                    if retry and result.to_dict()['has_error'] == 137:
                        message = f'Retry {result.case.repo}'
                        failed_tasks.append(taskmap[result.case.repo])
                    else:
                        message = f'[{finished + 1}/{total}] Processed {result.case.repo}'
                        self.summary.append(result.to_dict())
                        finished += 1

                    print(message, flush=True)
                except Exception as e:
                    print("Exception:", e, flush=True)

                except KeyboardInterrupt:
                    with WorkerLock:
                        for proc in SubProcSet.values():
                            proc.kill()
                    break

        return failed_tasks

    
    def run_bench_parallel(self, versions, debug, extra_flags):
        for v in versions:
            command = cleaner_install_command(v)
            for i in range(1, 4):
                _, executed = exec_docker_command_parallel(i, v, command, debug)
                print(executed)

        tasks = []
        container = 1
        for v in versions:
            for case in self.get_repos_version(v):
                if container > 2:
                    container = 1

                tasks.append(BenchTask(container, case, v, extra_flags, debug))
                container += 1
        
        self.summary = []

        print('Running {} tasks'.format(len(tasks)), flush=True)
        failed_tasks = self.run_bench_with_tasks(tasks, self.jobs, 0, len(tasks))

        if len(failed_tasks) > 0:
            failed_tasks = self.run_bench_with_tasks(failed_tasks, 8, len(failed_tasks), len(tasks))
        
        if len(failed_tasks) > 0:
            failed_tasks = self.run_bench_with_tasks(failed_tasks, 2, len(failed_tasks), len(tasks), False)
        
    def write_summary(self, output):
        errs = [s for s in self.summary if s['has_error']]
        if len(errs):
            print('The following repositories failed:')
            for s in errs:
                print(s['go_version'], '\t', s['repo'])

        with open(output, 'w') as fp:
            fp.write(json.dumps(self.summary) + '\n')

        print('Details was written into ', output)
