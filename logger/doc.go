
/*
  Package logger contains filters and handles for the logging utilities in Revel.
  These facilities all currently use the logging library called log15 at
  https://github.com/inconshreveable/log15

  Wrappers for the handlers are written here to provide a kind of isolation layer for Revel
  in case sometime in the future we would like to switch to another source to implement logging

  */
package logger
