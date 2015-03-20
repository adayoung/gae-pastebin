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

  /* http://www.developerdrive.com/2013/07/using-jquery-to-add-a-dynamic-back-to-top-floating-button-with-smooth-scroll/ */
  var offset = 220;
  var duration = 500;
  jQuery(window).scroll(function() {
      if (jQuery(this).scrollTop() > offset) {
          jQuery('.back-to-top').fadeIn(duration);
      } else {
          jQuery('.back-to-top').fadeOut(duration);
      }
  });

  jQuery('.back-to-top').click(function(event) {
      event.preventDefault();
      jQuery('html, body').animate({scrollTop: 0}, duration);
      return false;
  })
});

(function(i,s,o,g,r,a,m){i['GoogleAnalyticsObject']=r;i[r]=i[r]||function(){
(i[r].q=i[r].q||[]).push(arguments)},i[r].l=1*new Date();a=s.createElement(o),
m=s.getElementsByTagName(o)[0];a.async=1;a.src=g;m.parentNode.insertBefore(a,m)
})(window,document,'script','//www.google-analytics.com/analytics.js','ga');

ga('create', 'UA-44889074-1', 'ada-young.appspot.com');
ga('require', 'displayfeatures');
ga('send', 'pageview');