"""轻量 JWT 工具 - 通过 AI PaaS CLI 获取用户 JWT token，无外部依赖"""
import re
import shutil
import subprocess

_ansi_escape = re.compile(r"\x1b\[[0-9;]*m")
_jwt_pattern = re.compile(r"^[A-Za-z0-9-_]+\.[A-Za-z0-9-_]+\.[A-Za-z0-9-_]*$")

# site → JWT env 映射（i18n-bd 和 i18n 共用同一个 JWT env）
_SITE_TO_ENV: dict[str, str] = {
    "cn": "cn",
    "boe": "boe",
    "i18n-tt": "i18n",
    "i18n-bd": "i18n",
}


def get_user_jwt_token(site: str = None) -> str | None:
    """通过 AI PaaS CLI 获取用户 JWT token。

    Returns:
        JWT token 字符串，获取失败返回空字符串。
    """
    env = _SITE_TO_ENV.get(site, "cn") if site else None
    try:
        exec_args = [
            "npx", "-y",
            "--registry", "https://bnpm.byted.org",
            "@byted/aipaas@latest", "get-jwt",
        ]
        if env:
            exec_args.append("-e")
            exec_args.append(env)
        bin_name = exec_args[0]
        if shutil.which(bin_name) is None:
            raise Exception(f"{bin_name} is not installed. Please install it first.")

        token = ""
        result = subprocess.Popen(
            exec_args,
            stdout=subprocess.PIPE,
            stderr=subprocess.STDOUT,
            text=True,
        )

        for line in result.stdout:
            stripped = line.strip()
            clean = _ansi_escape.sub("", stripped)
            if _jwt_pattern.match(clean):
                token = clean
            else:
                print(line, end="")

        returncode = result.wait()
        if returncode != 0:
            raise subprocess.CalledProcessError(returncode, exec_args)
        return token
    except subprocess.CalledProcessError as e:
        print(f"获取 jwt token 失败（exit code {e.returncode}），用户未登录或jwt token过期。注意：用户登录或jwt token自动续期后都需要写文件，仅能在沙箱外执行成功；若不支持沙箱外执行或频繁失败，则直接提醒用户手动执行" + " ".join(exec_args))
        return ""
    except Exception as e:
        print(f"获取 jwt token 失败（exit code {getattr(e, 'returncode', 'unknown')}），用户未登录或jwt token过期。注意：用户登录或jwt token自动续期后都需要写文件，仅能在沙箱外执行成功；若不支持沙箱外执行或频繁失败，则直接提醒用户手动执行" + " ".join(exec_args))
        return ""
