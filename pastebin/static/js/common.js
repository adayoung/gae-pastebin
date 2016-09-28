$(document).ready(function(){
  $('.tagbox').bind('input propertychange', function(){
    var q = $(this).val().toLowerCase().replace(RegExp('[^a-z0-9 ]+', 'g'), '', 'g');

    q = q.split(' ');
    var p = [];

    for (var i=0; i < q.length; i++) {
      if (q[i].length <= 15 && p.length < 15) {
        if ($.inArray(q[i], p) < 0){
          p.push(q[i]);
        }
      }
    }

    $(this).val(p.join(' ').replace(RegExp('[ ]+', 'g'), ' ', 'g'));
  });
});

