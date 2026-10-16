"""theoryloop: a planted-mechanism benchmark for AI theory-building loops."""
from .theory import Proposition
from .worlds import make_world, WORLD_NAMES
from .loop import run, CONDITIONS

__all__ = ["Proposition", "make_world", "WORLD_NAMES", "run", "CONDITIONS"]
