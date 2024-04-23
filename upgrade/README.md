# upgrade

A tool used to determine the next semantic version.

## The situation that requires updating the Major Version

**The prerequisite for upgrading the Major Version number is that your version number is greater than or equal to `v1.0.0`. 
Otherwise, even if it is a breaking change, only the Minor Version number will be upgraded.**

1. You deleted public types, functions, and global variables;
2. You deleted public fields in the structure;
3. You changed the types of public struct fields;
4. You modified the generic parameters in the public type definitions (added, removed, modified, shifted);
5. You modified the public interface definitions (any modification other than parameter names counts);
6. You modified the public function signatures (any modification other than parameter names counts);
7. You changed the receivers of public methods from a pointer type to a value type;
8. You moved public types, functions, global variables under a certain build tag to another build tag (not specifying a 
   build tag is also considered a type of build tag).

*Note that constants are also considered as a type of variable here.*

## The situation that requires updating the Minor Version

1. You added public types, functions, and variables.
2. You added new public fields to the structure.
3. You modified the tag of the structure.

## The situation that requires updating the Patch Version

All other changes not listed above.