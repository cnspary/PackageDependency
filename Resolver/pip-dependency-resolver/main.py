import sys
from extras_provider import ExtrasProvider
from pip._vendor.packaging.requirements import Requirement
from pypi_wheel import PyPIProvider, BaseReporter, Resolver, display_resolution

def main():
    """Resolve requirements as project names on PyPI.
    The requirements are taken as command-line arguments
    and the resolution result will be printed to stdout.
    """
    if len(sys.argv) == 1:
        print(f"Usage: {sys.argv[0]} <PyPI project name(s)>")
        return

    reqs = sys.argv[1:]
    requirements = [Requirement(r) for r in reqs]
    user_requested = {a.name: i for i, a in enumerate(requirements)}

    provider = PyPIProvider(user_requested)
    reporter = BaseReporter()
    resolver = Resolver(provider, reporter)

    print(f"Resolving {', '.join(reqs)}")
    result = resolver.resolve(requirements)
    vpin, vedge = display_resolution(result)

    print(f"{vpin}\n")
    print(f"{vedge}\n")

if __name__ == "__main__":
    main()
