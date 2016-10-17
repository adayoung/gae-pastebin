$(document).ready(function(){
  $('input[type="text"]').on('focus', function(){
    $(this).select();
  });

  $(".copybtn").tooltip({container: 'body'});

  var clipboard = new Clipboard('.copybtn');
  clipboard.on('success', function(e) {
    $($(e.trigger).siblings()[1]).tooltip({
      placement: 'auto',
      title: 'Copied!',
      container: 'body'
    });
    $($(e.trigger).siblings()[1]).tooltip('show');
    setTimeout(function() {
      try {
        $($(e.trigger).siblings()[1]).tooltip('destroy');
      } catch(e) {};
    }, 1000);
  });

  clipboard.on('error', function(e) {
    $($(e.trigger).siblings()[1]).tooltip({
      placement: 'auto',
      title: 'Control-C to copy!',
      container: 'body'
    });
    $($(e.trigger).siblings()[1]).tooltip('show');
    setTimeout(function() {
      try {
        $($(e.trigger).siblings()[1]).tooltip('destroy');
      } catch(e) {};
    }, 1000);
  });
});
