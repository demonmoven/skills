class TestCase:
    def __init__(self, repo, head, version, description, build_flags='', cleaner_flags=''):
        self.repo = repo
        self.head = head
        self.version = version
        self.description = description
        self.build_flags = build_flags
        self.cleaner_flags = cleaner_flags
    
    def get_group(self):
        return self.repo.split('/')[0]
    
    def get_project(self):
        return self.repo.split('/')[1]

    def get_cleaner_command(self, extra_flags):
        executes = [
            'cd /tmp',
            f'rm -rf {self.get_project()}',
            f'rm -rf {self.repo}',
            f'mkdir -p {self.repo}',
            f'git clone git@code.byted.org:{self.repo} {self.repo}',
            f'cd {self.repo}',
            f'git checkout {self.head}',
            'timeout 40m cleaner -f --before=0',
        ]

        command = ' && '.join(executes)
        if self.build_flags:
                command += f' --build_flags="{self.build_flags}"'

        if self.cleaner_flags:
            command += ' ' + self.cleaner_flags
        
        if extra_flags:
            command += ' ' + ' '.join(extra_flags)

        return command

    def report_error(self, index, total):
        return f'\n\033[31m[{index}/{total}] {self.repo} (failed): {self.description}\033[0m'
    
    def report_start(self, index, total):
        return f'\n\033[33m[{index}/{total}] {self.repo}: {self.description}\033[0m'

class TestResult:
    def __init__(self, case):
        self.case = case
        self.messages = []
        self.proc = None
    
    def set_proc(self, proc):
        self.proc = proc
        if proc:
            self.add_message(proc.stdout)
            self.add_message(proc.stderr)
    
    def add_message(self, message):
        self.messages.append(message.strip())

    def has_error(self):
        return self.proc and self.proc.returncode

    def to_dict(self):
        msgs = []
        for msg in self.messages:
            msgs.extend([m.strip() for m in msg.split('\n') if m])

        return {
            'repo': self.case.repo,
            'go_version': self.case.version,
            'messages': self.messages,
            'has_error': self.has_error(),
        }