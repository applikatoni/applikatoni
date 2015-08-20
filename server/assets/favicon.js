var Favicon = (function() {

  var faviconSize = 32;

  var interval;

  var angle = 0;

  var linkEl;

  var img = new Image();

  var framesPerSecond = 10;

  var rotationPerFrame = 4; // degrees

  var canvas = document.createElement('canvas');
  canvas.width = canvas.height = faviconSize;

  var ctx = canvas.getContext('2d');

  function drawOnCanvas(angle) {
    ctx.clearRect(0, 0, faviconSize, faviconSize);

    ctx.save();
    ctx.translate(faviconSize / 2, faviconSize / 2);
    ctx.rotate(angle * Math.PI / 180);
    ctx.drawImage(img, -img.width / 2, -img.height / 2);
    ctx.restore();

    linkEl.attr({ href: canvas.toDataURL('image/png') });
  }

  function _startRotation() {
    stopRotation();

    interval = setInterval(function() {
      if (document.hidden) {
        // when tab is in background, setInterval will be called
        // max 1 time per second, so we adjust the rotation
        angle += rotationPerFrame * framesPerSecond;
      } else {
        angle += rotationPerFrame;
      }

      drawOnCanvas(angle);
    }, 1000 / framesPerSecond);
  }

  function startRotation() {
    linkEl = $('link[rel=icon]');


    img.onload = _startRotation;
    img.src = linkEl.attr('href');
  }

  function stopRotation() {
    return interval && clearInterval(interval);
  }

  return {
    startRotation: startRotation,
    stopRotation: stopRotation
  };

})();
