const {Landns, Record} = require('./dist');

const l = new Landns();
l.set([Record.parse("example.com. 123 IN A 127.0.0.1")]);
l.get().then(xs => {
    console.log(xs.map(x => x.toString()));
});
