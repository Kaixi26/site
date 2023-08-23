--
Title: Jumping towards fibers
Date: 22/08/2023
--

# Jumping towards fibers

I have been curious for a long time about how async systems work under the
hood, I understand them at a higher level and can use them pretty well, but I
have no idea how to start implementing one by myself, starting from the level
of the machine.

Searching for [fibers](https://en.wikipedia.org/wiki/Fiber_(computer_science)) on wikipedia lead me
to [getcontext](https://man7.org/linux/man-pages/man3/getcontext.3.html) and friends.

> "Less support from the operating system is needed for fibers than for threads.
> They can be implemented in modern Unix systems using the library functions
> getcontext, setcontext and swapcontext in ucontext.h, as in GNU Portable
> Threads, or in assembler as boost.fiber."

This in turn led me to their building blocks,
[setjmp](https://man7.org/linux/man-pages/man3/longjmp.3.html) and
[longjmp](https://man7.org/linux/man-pages/man3/longjmp.3p.html).
This is interesting, it provides a mechanism for "nonlocal gotos", we somehow
save the state of a function in this `jmp_buf` variable and can then jump back
to it.

## Jumping small

The man page for `setjmp` mentions registers so this is a good indication that
we'll have to dive into assembly, I will be using zigs [global
assembly](https://ziglang.org/documentation/0.11.0/#Global-Assembly) and arm64
architecture since that's what I have on the computer I'm writing this on. This
is my first time even looking at arm64 assembly so it's going to be a great
learning opportunity.

```zig
const std = @import("std");

comptime {
    asm (
        \\_inc:
        \\  ldr x9, =1
        \\  add x0, x0, x9
        \\  ret
    );
}

extern fn inc(usize) usize;

pub fn main() void {
    std.debug.print("{}\n", .{inc(68)});
}
```

```sh
$ zig run main.zig
69
```

Awesome, so how does this work?
Here is an [arm64
cheatsheet](https://www.cs.swarthmore.edu/~kwebb/cs31/resources/ARM64_Cheat_Sheet.pdf)
and a description of [procedure call
standard](https://developer.arm.com/documentation/102374/0101/Procedure-Call-Standard).

The label `inc` was resulting in a linking error so I had to change it to `_inc`.
My understanding is that this is because C compilers usually do this to their
names, and since extern will generally be used to call C functions the `_` is
implicitly added, probably there's something to override this behaviour but I
did not check.
You can verify that the C compilers do this:

```c
void foo() {}
```

```sh
$ clang -c foo.c
$ nm foo.o
0000000000000000 T _foo
0000000000000000 t ltmp0
0000000000000008 s ltmp1
```

The instruction `ldr x9, =1` loads the value 1 into register `x9`, this
register is corruptable so the callee cannot depend on it being the same after
the function call.
After that we do `add x0, x0, x9`, notice that the first argument is always the
destination, this adds the contents of registers x0 and x9.
Registers `x0-x7` are by convention used to pass around the function
parameters, thus, the value of `x0` is the first argument to the function.

Finally, we have `ret` that jumps back to the instruction after the call.
To be more precise, it performs the computation `PC = x30`, so this `x30` is
the register that stores the place we must return to, which of course makes you
wonder, what if we update it to jump to a different place?

```zig
const std = @import("std");

comptime {
    asm (
        \\_set_jump:
        \\  str x30, [x0]
        \\  ldr x0, =0
        \\  ret
        \\
        \\_long_jump:
        \\  ldr x30, [x0]
        \\  mov x0, x1
        \\  ret
    );
}

const JumpBuffer = packed struct {
    x30: usize,
};
extern fn set_jump(*JumpBuffer) usize;
extern fn long_jump(*JumpBuffer, usize) noreturn;

pub fn main() void {
    var jump_buffer: JumpBuffer = undefined;
    const val = set_jump(&jump_buffer);
    std.debug.print("val: {}\n", .{val});
    while (val < 3) {
        long_jump(&jump_buffer, val + 1);
    }
}
```

```sh
zig run main.zig
val: 0
val: 1
val: 2
val: 3
```

Awesome, we are gods, going back in time and modifying a `const` variable.
We simply store the address of the instruction after `set_jump` into the
`JumpBuffer` and then when we run `long_jump` we override `x30` with this saved
address, we also update `x0` so now `val` will be this "fake" return value.
Now let's try a bigger jump!

```zig
pub fn main() void {
    var jump_buffer: JumpBuffer = undefined;
    const val = set_jump(&jump_buffer);
    std.debug.print("val: {}\n", .{val});
    while (val < 3) {
        jumpFromAbove(&jump_buffer, val + 1);
    }
}

pub fn jumpFromAbove(bj: *JumpBuffer, val: usize) noreturn {
    long_jump(bj, val);
}
```

```sh
$ zig run -OReleaseFast main.zig                                                                                                     ✘ 130
val: 0
val: 1
zsh: segmentation fault  zig run -OReleaseFast main.zig
```

Uh oh, what the hell happened here?
Somehow trying to jump when we are inside of the other function is causing a
segfault.
Notice that `val` gets printed twice though, so clearly it's working the first
time but somehow the second is getting corrupted.
Let's check [godbolt](https://godbolt.org/) and try to see what is happening
when calling the function.
```zig
extern fn long_jump() noreturn;
export fn foo() noreturn {
    long_jump();
}
```

```arm64
_foo:
        stp     x29, x30, [sp, #-16]!
        mov     x29, sp
        bl      _long_jump
```

Oh, of course we are moving the stack pointer and not changing it back. 
This means on the second "fake" return we won't be looking for `jump_buffer` in
the correct address, causing us to jump into an undefined place and crash.
Let's try to fix this by also storing and restoring both the stack pointer (sp)
and the frame pointer (x29).

```zig
comptime {
    asm (
        \\_set_jump:
        \\  str x29, [x0, 0]
        \\  str x30, [x0, 8]
        \\  mov x9, sp
        \\  str x9,  [x0, 16]
        \\
        \\  ldr x0, =0
        \\  ret
        \\
        \\_long_jump:
        \\  ldr x29, [x0, 0]
        \\  ldr x30, [x0, 8]
        \\  ldr x9,  [x0, 16]
        \\  mov sp, x9
        \\
        \\  mov x0, x1
        \\  ret
    );
}

const JumpBuffer = packed struct {
    x29: usize,
    x30: usize,
    sp: usize,
};
```

```sh
zig run -OReleaseFast main.zig
val: 0
val: 1
val: 2
val: 3
```

It seems to be working now, awesome.
It took me a bit to figure out why sp wasn't working properly, I was using x31
as I saw in the cheatsheet.
However, it seems that in arm64 this register is special and has a double
meaning, in certain cases it refers to `sp` but in others it refers to `0`,
this means that we can't direcly do `str x31, [x0, 16]` as this would store the
value `0` instead of the value of `sp`.
Apparently this is done in a way to be able to [reduce complexity of the
instruction
set](https://stackoverflow.com/questions/52410521/is-zero-register-zr-in-aarch64-essentially-ground).

We have one final detail to take care of, we are not storing registers
`x19-x28` and `d8-d15`.
These are callee-saved registers which means if a function modifies them they
have to return them to their original state after.
This could happen for example in the call for `jumpFromAbove`, which would mean
we jump back with bad registers.
To do this is simple, we can simply do the same we did with `x29` and `x30`.
In addition, there is an optimization we can do using `stp` which takes two
registers at the same time and stores them into memory in a 128bit region,
`ldp` does the opposite operation.

```
stp S1, S2, [R]
  Mem[R] = S1
  Mem[R + 8] = S2

ldp D1, D2, [R]
  D1 = Mem[R]
  D2 = Mem[R + 8]
```

We can even use the power of zig comptime to generate the code for us.
This is nice since I failed when I tried to write the assembly with counting
the offsets in my head.
I don't know how they did this in the past, maybe they failed a lot or maybe
they are just smarter than me.

```zig
pub fn main() void {
    var jump_buffer: JumpBuffer = undefined;
    const fields = @typeInfo(JumpBuffer).Struct.fields;
    std.debug.print("_set_jump:\n", .{});
    inline for (fields) |field| {
        const ptri = @intFromPtr(&jump_buffer);
        const ptre = @intFromPtr(&@field(jump_buffer, field.name));
        if (std.mem.eql(u8, field.name, "sp")) {
            std.debug.print("mov x9, {s}\nstr x9, [x0, {}]\n", .{ field.name, ptre - ptri });
        } else {
            std.debug.print("str {s}, [x0, {}]\n", .{ field.name, ptre - ptri });
        }
    }
    std.debug.print("\n_long_jump:\n", .{});
    inline for (fields) |field| {
        const ptri = @intFromPtr(&jump_buffer);
        const ptre = @intFromPtr(&@field(jump_buffer, field.name));
        if (std.mem.eql(u8, field.name, "sp")) {
            std.debug.print("ldr x9, [x0, {}]\nmov {s}, x9\n", .{ ptre - ptri, field.name });
        } else {
            std.debug.print("ldr {s}, [x0, {}]\n", .{ field.name, ptre - ptri });
        }
    }
}
```

This should get us to a safe enough implementation, however, it doesn't match
the size of the `jmp_buf`.
Looking at a reference implementation of
[setjmp](https://chromium.googlesource.com/external/github.com/WebAssembly/musl/+/wasm-prototype-1/src/setjmp/aarch64/setjmp.s)
it seems that some of the extra space is left unused, I assume this is for
compatibility reasons although I am not sure.

Here is the final code.
```zig
const std = @import("std");

comptime {
    asm (
        \\_set_jump:
        \\  stp x19, x20, [x0, 0]
        \\  stp x21, x22, [x0, 16]
        \\  stp x23, x24, [x0, 32]
        \\  stp x25, x26, [x0, 48]
        \\  stp x27, x28, [x0, 64]
        \\  stp x29, x30, [x0, 80]
        \\
        \\  mov x9, sp
        \\  str x9, [x0, 96]
        \\
        \\  stp d8, d9, [x0, 104]
        \\  stp d10, d11, [x0, 120]
        \\  stp d12, d13, [x0, 136]
        \\  stp d14, d15, [x0, 152]
        \\
        \\  ldr x0, =0
        \\  ret
        \\
        \\_long_jump:
        \\  ldp x19, x20, [x0, 0]
        \\  ldp x21, x22, [x0, 16]
        \\  ldp x23, x24, [x0, 32]
        \\  ldp x25, x26, [x0, 48]
        \\  ldp x27, x28, [x0, 64]
        \\  ldp x29, x30, [x0, 80]
        \\
        \\  ldr x9, [x0, 96]
        \\  mov sp, x9
        \\
        \\  ldp d8, d9, [x0, 104]
        \\  ldp d10, d11, [x0, 120]
        \\  ldp d12, d13, [x0, 136]
        \\  ldp d14, d15, [x0, 152]
        \\
        \\  mov x0, x1
        \\  ret
    );
}

const JumpBuffer = packed struct {
    x19: usize,
    x20: usize,
    x21: usize,
    x22: usize,
    x23: usize,
    x24: usize,
    x25: usize,
    x26: usize,
    x27: usize,
    x28: usize,
    x29: usize,
    x30: usize,
    sp: usize,
    d8: usize,
    d9: usize,
    d10: usize,
    d11: usize,
    d12: usize,
    d13: usize,
    d14: usize,
    d15: usize,
};
extern fn set_jump(*JumpBuffer) usize;
extern fn long_jump(*JumpBuffer, usize) noreturn;

pub fn main() void {
    std.debug.print("size: {}\n", .{@sizeOf(JumpBuffer)});
    var jump_buffer: JumpBuffer = undefined;
    const val = set_jump(&jump_buffer);
    std.debug.print("val: {}\n", .{val});
    while (val < 3) {
        long_jump(&jump_buffer, val + 1);
    }
}
```
