"""
Setup script for building Cython extensions
Run: python3 setup_cython.py build_ext --inplace
"""

from setuptools import setup, Extension
from Cython.Build import cythonize
import numpy

extensions = [
    Extension(
        "cpu_cython",
        ["src_python/cpu_cython.pyx"],
        include_dirs=[numpy.get_include()],
        extra_compile_args=["-O3", "-ffast-math"],  # Aggressive optimization
        extra_link_args=["-O3"],
    ),
]

setup(
    name="Nitro-Core-DX",
    ext_modules=cythonize(
        extensions,
        compiler_directives={
            "language_level": "3",
            "boundscheck": False,
            "wraparound": False,
            "cdivision": True,
            "optimize.use_switch": True,
        },
        annotate=True,  # Generate HTML annotation file for optimization
    ),
)

