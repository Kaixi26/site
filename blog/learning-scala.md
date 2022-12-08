--
Title: Learning scala
Date: 04/12/2022
--

# Learning Scala

Recently I've been learning [Scala](https://www.scala-lang.org/), it is an
interesting language.
It has a lot of the same feel of languages like haskell, however, it also feels
very pragmatic at the same time.
I can't fully describe it yet.

I'm gonna use this article to save features I have learned about, for later
reference.
And hopefully will keep it updated as I learn more.

Some useful links:
- [Scala cheatsheet](https://docs.scala-lang.org/cheatsheets/index.html)

## Scala

### Pattern Matching

Pattern matching is done using the `unapply` and `unapplySeq` methods.
Case classes already define this method automatically, however it is also
possible to define this method by hand.

The `match` keyword is used the following way:

```scala
val x = 1
x match {
  case 1 => println("x = 1") // x will match this case
}
```

It is also possible to add conditions for the `case` statement:
```scala
val x = 1
x match {
  case x if x % 2 == 0 => "even"
  case x => "odd"
}
```

This is an example of using the `unapply` method to create a custom match.

```scala
object MatchDoubler {
  def unapply(x: Int): Option[Int] = Some(x * 2)
}
val x = 1
x match {
  case MatchDoubler(y) => println("y = 2") // x will match this case
}
```

This is an example of using the `unapplySeq` method to create a custom match.

```scala
object MatchDuplicator {
  def unapplySeq(x: Any): Option[Seq[Any]] = Some(Seq(x,x))
}
val x = 1
x match {
  case MatchDuplicator(y,z) => println("y = 1, z = 1") // x will match this case
}
```

It is also possible to use `case` directly on a lambda with one argument, it
will match on that argument.

```scala
Seq(1,2,3).map {
  case 1 => 2
  case x => x
}
```

### Variance
Variance lets you control how type parameters behave with regards to subtyping.[(from docs)](https://docs.scala-lang.org/tour/variances.html)

#### Invariance

This is the default behaviour, the type parameter has to match exactly.

```scala
class X
class Y extends X
class C[A]

object Main extends App {
  val x: C[X] = new C[X]
  val y: C[X] = new C[Y] // fails to compile
}
```

#### Covariance

This allows subtypes to be treated as a more generic type.
The following code compiles without problems.

```scala
class X
class Y extends X
class C[+A]

object Main extends App {
  val x: C[X] = new C[X]
  val y: C[X] = x
}
```

#### Contravariance

This allows a generic type to be treated as a more specific type.
Imagine a class `Adder[Number]` that can add any numbers, it makes sense that
this class can also be used as an `Adder[Int]`.
The following code compiles without problems.

```scala
class X
class Y extends X
class C[-A]

object Main extends App {
  val x: C[Y] = new C[Y]
  val y: C[Y] = x
}
```

#### Notes

Interestingly, if a type could be variant and contravariant at the same time, it
would mean you would be able to treat it as any other type.

```scala
class X
class Y extends X
class Z extends X
class C[+-A]

object Main extends App {
  val y: C[Y] = new C[Y]
  val x: C[X] = y
  val z: C[Z] = x
}

```

### Functors & Monads

```scala
case class Identity[A](v: A) {
  def map(f: A => A): Identity[A] = { Identity(f(v)) }
  def flatMap(fb: A => Identity[A]): Identity[A] = { fb(v) }
}

object Main extends App {
  
  val x = for {
    x <- Identity(1)
    y <- Identity(x * 2)
  } yield x + y
  
  val y = Identity(1).flatMap(x => {
    Identity(x * 2).flatMap(y => {
      Identity.apply(x + y)
    })
  })
  println(x == y)
}
```
