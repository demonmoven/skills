#!/usr/bin/env python3
"""
SCM Build Tool - CLI tool for creating and monitoring SCM build tasks

Usage examples:
    python main.py --repo code_forge/pipeline/worker --branch dev-zzx-metrics --type offline
    python main.py -r my_repo -b my_branch -t online
    python main.py -r my_repo --version 3.0.4.4209

Features:
1. Create SCM build version
2. Poll build status
3. Display build results or error logs
"""
import argparse
import asyncio
import logging

import bytedlogger
bytedlogger.config_default()

from internal.byte_build_client import BytebuildNightlyClient
from internal.scm_client import SCMClient
from internal.scm_auth import set_token, decode_jwt_username
from internal import models

# Global default configuration
MAX_ATTEMPTS = 60
POLL_INTERVAL = 30


def parse_arguments():
    """Parse command line arguments"""
    parser = argparse.ArgumentParser(
        description="SCM Build Tool - Create and monitor SCM build tasks",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Usage examples:
  %(prog)s -r code_forge/pipeline/worker -b dev-zzx-metrics -t offline
  %(prog)s --repo my_repo --branch my_branch --type online
  %(prog)s -r repo -b branch -t offline
  %(prog)s -r repo --version 3.0.4.4209

Notes:
  - Default poll attempts: 60 (fixed)
  - Default poll interval: 30 seconds (fixed)
  - Build status query waits 30 seconds for initialization after version creation (fixed)
        """
    )

    parser.add_argument(
        "--jwt",
        help="Personal JWT token for authentication (overrides SCM_JWT_TOKEN env var)"
    )

    parser.add_argument(
        "-r", "--repo",
        required=True,
        help="Repository name (e.g., code_forge/pipeline/worker)"
    )

    branch_or_version = parser.add_mutually_exclusive_group(required=True)
    branch_or_version.add_argument(
        "-b", "--branch",
        help="Branch name (e.g., master)"
    )
    branch_or_version.add_argument(
        "--version",
        help="Version number (e.g., 3.0.4.4209)"
    )

    parser.add_argument(
        "-t", "--type",
        choices=["online", "offline", "test"],
        default="offline",
        help="Build type: online, offline, or test (default: offline)"
    )

    parser.add_argument(
        "--user",
        default=None,
        help="Creator user (default: auto-detected from JWT token)"
    )

    parser.add_argument(
        "--verbose", "-v",
        action="store_true",
        help="Enable verbose logging output"
    )

    return parser.parse_args()


def resolve_user(args: argparse.Namespace) -> str:
    """Resolve the create_user from --user flag or JWT payload.

    Priority: --user flag > JWT payload > error
    """
    if args.user:
        return args.user

    username = decode_jwt_username()
    if username:
        logging.info(f"👤 Auto-detected user from JWT: {username}")
        return username

    logging.warning("⚠️ Could not detect username from JWT, falling back to 'root'")
    return "root"


async def cli(args: argparse.Namespace) -> str:
    """
    Execute SCM build task main workflow

    Args:
        args: Command line argument object

    Returns:
        str: Build result information
    """
    create_user = resolve_user(args)

    async with SCMClient() as client, BytebuildNightlyClient(
        retries=5,
        retry_statuses={404},
        retry_backoff_base=2,
        retry_backoff_max=30,
        retry_jitter=0.25,
    ) as bytebuild_client:
        if args.version:
            logging.info(f"🔎 Using existing version {args.version} to query build status")
            version_id = "unknown"
            version_number = args.version
            repo_id = None
        else:
            logging.info(f"🚀 Starting {args.type} build task for repository {args.repo} branch {args.branch} (user: {create_user})")

            # Step 1: Create SCM build version
            logging.info("📦 Creating SCM build version...")
            try:
                response = await client.cicd_create(models.CicdCreateRequest(
                    repo_name=args.repo,
                    branch_name=args.branch,
                    type=args.type,
                    create_user=create_user
                ))
                logging.info("✅ SCM version created successfully")
            except Exception as e:
                logging.error(f"❌ Failed to create SCM version: {e}")
                raise

            # Wait for build initialization
            logging.info("⏳ Waiting 30 seconds for SCM build process initialization...")
            await asyncio.sleep(30)

            # Extract version information
            version_id = str(response.context.version_id)
            version_number = response.context.version_version
            repo_id = response.context.repo_id

            logging.info(f"📋 Created SCM version {version_number}, ID: {version_id}")

        # Get repo_id if not in response
        if not repo_id:
            logging.info(f"🔍 Getting repository ID for {args.repo}...")
            try:
                get_repo_response = await client.repos_by_names(repo_names=[args.repo])
                if args.verbose:
                    logging.info(f"📄 Repository response: {get_repo_response.model_dump_json(indent=2)}")
                repo_id = get_repo_response.repos[0].id
                logging.info(f"📍 Repository {args.repo} ID: {repo_id}")
            except Exception as e:
                logging.error(f"❌ Failed to get repository ID: {e}")
                raise

        # Step 2: Poll build status
        logging.info(f"🔄 Starting to poll build status for version {version_number}...")

        for attempt in range(MAX_ATTEMPTS):
            try:
                version_result = await client.get_version(
                    repo_id=repo_id,
                    version_name=version_number
                )
                status = version_result.status
                status_display = version_result.status_display

                logging.info(f"📊 Poll {attempt + 1}/{MAX_ATTEMPTS}: "
                           f"Status = {status} ({status_display})")

                if not version_result.is_terminal():
                    logging.info(f"⌛ Build in progress, waiting {POLL_INTERVAL} seconds...")
                    await asyncio.sleep(POLL_INTERVAL)
                    continue

                # Build successful
                if version_result.is_build_ok():
                    success_msg = version_result.get_build_ok_string(version_id=version_id)
                    logging.info("🎉 Build completed successfully!")
                    return success_msg

                # Build failed - try to get detailed logs
                if version_result.is_build_failed():
                    build_num = version_result.get_build_num()
                    bytebuild_step = (version_result.failed_step.replace("_", "-") if version_result.failed_step else None)

                    # Retry if build_num or failed_step not available
                    retry_times = 5
                    if not build_num or not bytebuild_step:
                        for retry in range(retry_times):
                            logging.info(f"⚠️ Could not get build_num/failed_step, retrying ({retry + 1}/{retry_times})...")
                            backoff = min(2 * (2 ** retry), 10)
                            await asyncio.sleep(backoff)
                            try:
                                version_result = await client.get_version(
                                    repo_id=repo_id,
                                    version_name=version_number
                                )
                                build_num = version_result.get_build_num()
                                bytebuild_step = (version_result.failed_step.replace("_", "-") if version_result.failed_step else None)
                                if build_num and bytebuild_step:
                                    logging.info(f"✅ Got build_num: {build_num}, step: {bytebuild_step}")
                                    break
                            except Exception as e:
                                logging.warning(f"⚠️ Retry failed: {e}")
                    if build_num and bytebuild_step:
                        logging.info(f"📋 Getting failure logs for build number {build_num}...")
                        try:
                            build_logs_response = await bytebuild_client.get_building_logs(
                                record_id=build_num,
                                step_name=bytebuild_step
                            )
                            failed_msg = version_result.get_build_failed_string_with_logs(
                                version_id=version_id,
                                build_logs=build_logs_response.logs
                            )
                            logging.error("💥 Build failed (detailed logs retrieved)")
                            return failed_msg
                        except Exception as e:
                            logging.warning(f"⚠️ Failed to get build logs: {e}")

                    # Return basic failure info when logs unavailable
                    failed_msg = version_result.get_build_failed_string_without_logs(
                        version_id=version_id
                    )
                    logging.error("💥 Build failed (no detailed logs)")
                    return failed_msg

            except Exception as e:
                logging.error(f"❌ Error during poll attempt {attempt + 1}: {e}")
                if attempt == MAX_ATTEMPTS - 1:
                    raise
                await asyncio.sleep(POLL_INTERVAL)
                continue

        # Step 3: Handle timeout
        error_msg = f"Build not completed after {MAX_ATTEMPTS} poll attempts (total: {MAX_ATTEMPTS * POLL_INTERVAL} seconds)"
        logging.error(f"⏰ {error_msg}")
        return version_result.get_build_timeout_string(
            version_id=version_id,
            max_poll_attempts=MAX_ATTEMPTS,
            poll_interval=POLL_INTERVAL
        )



async def main():
    """Main entry point"""
    try:
        args = parse_arguments()

        # Set verbose logging level
        if args.verbose:
            logging.getLogger().setLevel(logging.DEBUG)
            logging.debug("Verbose logging mode enabled")

        # Set JWT token if provided via --jwt flag
        if args.jwt:
            set_token(args.jwt)

        result = await cli(args)
        logging.info(f"🏁 Task completed: {result}")
        return 0

    except KeyboardInterrupt:
        logging.info("⚠️ User interrupted operation")
        return 1
    except SystemExit:
        # argparse calls sys.exit()
        return 1
    except Exception as e:
        logging.error(f"💥 Program execution failed: {e}")
        if logging.getLogger().level <= logging.DEBUG:
            import traceback
            logging.debug(f"Detailed error traceback:\n{traceback.format_exc()}")
        return 1


if __name__ == "__main__":
    asyncio.run(main())
