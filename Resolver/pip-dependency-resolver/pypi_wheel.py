import sys
import collections
import math
from packaging.utils import canonicalize_name
from pip._vendor.resolvelib.structs import DirectedGraph
from resolvelib import BaseReporter, Resolver
from extras_provider import ExtrasProvider
from sqlclass import *
from pip._vendor.packaging.requirements import Requirement
from pip._vendor.packaging.specifiers import SpecifierSet
from pip._vendor.packaging.version import Version, LegacyVersion
from model import database
from platform import python_version
from operator import attrgetter
import copy

PYTHON_VERSION = str(Version(python_version()))

Result = collections.namedtuple("Result", "mapping graph criteria")


def _has_route_to_root(criteria, key, all_keys, connected):
    if key in connected:
        return True
    if key not in criteria:
        return False
    for p in criteria[key].iter_parent():
        try:
            pkey = all_keys[id(p)]
        except KeyError:
            continue
        if pkey in connected:
            connected.add(key)
            return True
        if _has_route_to_root(criteria, pkey, all_keys, connected):
            connected.add(key)
            return True
    return False


def _build_result(state):
    mapping = state.mapping
    all_keys = {id(v): k for k, v in mapping.items()}
    all_keys[id(None)] = None

    graph = DirectedGraph()
    graph.add(None)  # Sentinel as root dependencies' parent.

    connected = {None}
    for key, criterion in state.criteria.items():
        if not _has_route_to_root(state.criteria, key, all_keys, connected):
            continue
        if key not in graph:
            graph.add(key)
        for p in criterion.iter_parent():
            try:
                pkey = all_keys[id(p)]
            except KeyError:
                continue
            if pkey not in graph:
                graph.add(pkey)
            graph.connect(pkey, key)

    return graph


class Candidate:
    def __init__(self, name, spec_version, dependency=[], yanked=False, extras=None):
        self.name = canonicalize_name(name)
        self.spec_version = spec_version
        try:
            self.version = Version(spec_version)
        except:
            self.version = LegacyVersion(spec_version)
        self.extras = extras
        self.dependency = dependency
        self._dependencies = []
        self.parsed = False
        self.yanked = yanked

    def __repr__(self):
        if not self.extras:
            return f"<{self.name}=={self.version}>"
        return f"<{self.name}[{','.join(self.extras)}]=={self.version}>"

    @property
    def dependencies(self):
        if not self.parsed:
            self.parsed = True
            for a in self.dependency:
                try:
                    req_instance = Requirement(a)
                    req_instance.name = canonicalize_name(req_instance.name)
                except:
                    continue
                if req_instance.marker is None:
                    self._dependencies.append(req_instance)
                else:
                    for extra in self.extras:
                        if req_instance.marker.evaluate({"extra": extra, "python_version": PYTHON_VERSION}):
                            self._dependencies.append(req_instance)
                    if req_instance.marker.evaluate({"extra": "MC_checknomal_None", "python_version": PYTHON_VERSION}):
                        self._dependencies.append(req_instance)
        return self._dependencies


class PyPIProvider(ExtrasProvider):
    def __init__(self, user_requested):
        self._user_requested = user_requested
        self._known_depths = collections.defaultdict(lambda: math.inf)

    def identify(self, requirement_or_candidate):
        if isinstance(requirement_or_candidate, Requirement):
            if requirement_or_candidate.extras:
                formatted_extras = ",".join(set(requirement_or_candidate.extras))
                return f"{requirement_or_candidate.name}[{formatted_extras}]"
        return canonicalize_name(requirement_or_candidate.name)

    def get_extras_for(self, requirement_or_candidate):
        return tuple(sorted(requirement_or_candidate.extras))

    def get_base_requirement(self, candidate):
        return Requirement(f"{candidate.name}=={candidate.version}")

    def get_preference(self, identifier, resolutions, candidates, information, backtrack_causes):
        criterion = list(information[identifier])
        try:
            next(iter(information[identifier]))
        except StopIteration:
            has_information = False
        else:
            has_information = True

        candidate = None
        ireqs = [a.requirement for a in criterion]
        operators = [specifier.operator for specifier_set in (ireq.specifier for ireq in ireqs if ireq) for specifier in specifier_set]

        direct = candidate is not None
        pinned = any(op[:2] == "==" for op in operators)
        unfree = bool(operators)

        requested_order = self._user_requested.get(identifier, math.inf)
        inferred_depth = 1.0 if has_information else math.inf
        self._known_depths[identifier] = inferred_depth

        delay_this = identifier == "setuptools"

        return (
            delay_this,
            not direct,
            not pinned,
            inferred_depth,
            requested_order,
            not unfree,
            identifier,
        )

    def find_matches(self, identifier, requirements, incompatibilities):
        requirements = set(requirements[identifier])
        extras_set = {extra for r in requirements for extra in r.extras}

        bad_versions = {c.version for c in incompatibilities[identifier]}
        candidates = [
            candidate
            for candidate in self.get_project_from_pypi(canonicalize_name(r.name), extras_set)
            if candidate.version not in bad_versions
        ]

        allowed_candidates = []
        for req_obj in requirements:
            if req_obj.specifier == SpecifierSet(""):
                allowed_candidates.extend(c for c in candidates if not c.version.is_prerelease)
                candidates = allowed_candidates[:]
                allowed_candidates = []
            prereleases = any(spec.prereleases for spec in req_obj.specifier)
            for spec in req_obj.specifier:
                allowed_candidates.extend(c for c in candidates if spec.contains(c.version, prereleases=prereleases))
                candidates = allowed_candidates[:]
                allowed_candidates = []
        return sorted([c for c in candidates if not c.yanked or any(spec.operator == "==" for spec in req_obj.specifier)], key=attrgetter("version"), reverse=True)

    def is_satisfied_by(self, requirement, candidate):
        if canonicalize_name(requirement.name) != canonicalize_name(candidate.name):
            return False
        return requirement.specifier.contains(candidate.version, prereleases=True)

    def get_dependencies(self, candidate):
        return candidate.dependencies

    def get_project_from_pypi(self, project, extras):
        sql = f"select name, version, dependency, yanked from projects_metadata where name = '{canonicalize_name(project)}' order by version collate en_natural desc;"
        q = database.execute_sql(sql)
        return [Candidate(a[0], a[1], eval(a[2]) if a[2] else [], a[3], extras=extras) for a in q]


def display_resolution(result):
    graph = _build_result(result)
    vnode = copy.deepcopy(result.graph._vertices)
    vedge = copy.deepcopy(result.graph._forwards)
    vpin = {canonicalize_name(node): str(result.mapping[node].version) if node else '' for node in graph._vertices if node}

    for key in list(result.graph._vertices):
        if key and canonicalize_name(key) != key:
            vpin[canonicalize_name(key)] = vpin.pop(key)

    for key, value in list(vedge.items()):
        if key and canonicalize_name(key) != key:
            vedge[canonicalize_name(key)] = vedge.pop(key)
        for dep in value:
            if canonicalize_name(dep) != dep:
                vedge[canonicalize_name(key)].add(canonicalize_name(dep))
                vedge[canonicalize_name(key)].remove(dep)

    for key in vedge:
        if key and "[" in key:
            vedge[key].add(key.split("[")[0])

    return vpin, vedge


class DepParser:
    def __init__(self, reqs):
        self.reqs = reqs
        self.requirements = [
            req_instance for req in reqs if not req.startswith('#') and '#' not in req
            if (req_instance := Requirement(req)) and self.get_project_from_pypi(req_instance)
            and (not req_instance.marker or req_instance.marker.evaluate({"extra": "MC_checknomal_None"}))
        ]
        self.user_requested = {a.name: i for i, a in enumerate(self.requirements)}
        self.provider = PyPIProvider(self.user_requested)
        self.reporter = BaseReporter()
        self.resolver = Resolver(self.provider, self.reporter)

    def resolve(self, max_round):
        return self.resolver.resolve(self.requirements, max_round)

    def get_project_from_pypi(self, requirement: Requirement):
        sql = f"select name, version from projects_metadata where name = '{canonicalize_name(requirement.name)}' order by version collate en_natural desc;"
        q = database.execute_sql(sql)
        return [Candidate(a[0], a[1]) for a in q if requirement.specifier.contains(Candidate(a[0], a[1]).version, prereleases=True)]