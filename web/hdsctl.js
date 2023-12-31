function renderWaves(waves) {
    let c = document.getElementById("myCanvas");
    let w = c.width;
    let h = c.height;
    let hScale = 1;
    let wSmall = parseInt(w) / 12;
    let hSmall = parseInt(h) / 8;
    let ctx = c.getContext("2d");
    ctx.fillStyle = "black";
    ctx.fillRect(0, 0, w, h);
    ctx.fillStyle = "white";

    for (let j = 0; j < h; j++) {
        for (let i = 0; i < w; i++) {
            if ((i % (wSmall / 5) === 0 && (j % hSmall === 0 || j === h / 2 + 1 || j === h / 2 - 1 || j === h - 1)) ||
                (((i % wSmall === 0) || (i === w / 2 - 1) || (i === w / 2 + 1) || i === w - 1) && j % (hSmall / 5) === 0)) {
                ctx.fillRect(i, j, 1, 1);
            }
        }
    }

    for (var wav of waves) {
        ctx.beginPath();
        ctx.mozImageSmoothingEnabled = false;
        ctx.strokeStyle = wav.color;
        ctx.lineWidth = 2;
        let i = 0;
        let values = wav.data.split(" ");
        for (var v of values) {
            //let j = 115 - parseInt(v); // when using 230 maximum vertical resolution
            let j = Math.min(200, Math.max(1, 100 - parseInt(v)));
            if (i == 0) {
                ctx.moveTo(0, j);
            } else {
                ctx.lineTo(i * 300 / values.length, j);
                //ctx.lineTo(i, j *2);
            }
            i++;
        }
        ctx.stroke();
    }
}

window.addEventListener("load", function (evt) {
    let socket = new WebSocket("{{ .wsEndpoint }}");

    socket.onmessage = event => {
        fields = JSON.parse(event.data);
        waves = [];
        if (fields['wave1']) {
            waves.push({data: fields['wave1'], color: 'yellow'});
        }
        if (fields['wave2']) {
            waves.push({data: fields['wave2'], color: 'blue'});
        }
        if (waves.length>0) {
            renderWaves(waves);
        }
        for (var k in fields) {
            if (k.endsWith(".range")) {
                var parts = k.split(".");
                const dropDown = document.getElementById(parts[0]);
                dropDown.innerHTML = '';
                for (let v of fields[k]) {
                    let option = document.createElement("option");
                    option.setAttribute('value', v);
                    let optionText = document.createTextNode(v);
                    option.appendChild(optionText);
                    dropDown.appendChild(option);
                }
            }
        }
        for (var k in fields) {
            if (k === "wave1" || k === "wave2" || k.endsWith(".range")) {
                continue;
            }
            var value = fields[k];
            console.log("received: ", k, value);
            var el = document.getElementById(k);
            if (!el) {
                console.log("unknown element: ", k);
                continue;
            }
            if (["funcOffs","funcAmpl","funcLow", "funcHigh"].includes(k)) {
                el.value = (parseFloat(value)/1000).toFixed(2);
                continue;
            }
            if (["funcFreq"].includes(k)) {
                el.value = (parseFloat(value)/1000000).toFixed(3);
                continue;
            }
            el.value = value;
        }
    }
    var inputs = document.getElementsByTagName('input');
    for (var range of inputs) {
        range.addEventListener("keyup", evt => {
            if (evt.keyCode === 13) {
                socket.send(evt.target.id + ": " + evt.target.value);
            }
        });
    }
    inputs = document.getElementsByTagName('select');
    for (var range of inputs) {
        range.addEventListener("input", evt => {
            socket.send(evt.target.id + ": " + evt.target.value);
        });
    }

});