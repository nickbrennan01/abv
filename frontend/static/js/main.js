let persist = {};
persist.index = 0;

let timer = 5000;
let beersPerPage = 16;

function changePage(){
    $.getJSON("http://localhost:8081/inventory", function(beers){
        $("#beer-list").empty()
        if (persist.index >= beers.length) {
            persist.index = 0;
        }
        for (var i = persist.index; i < persist.index + beersPerPage; i++) {
            if (i == beers.length) {
                break;
            }
            var html = Mustache.to_html($('#beer-entry').html(), beers[i]);
            $('<div class="grid-item"/>').html(html).appendTo('#beer-list')
        }
        persist.index += beersPerPage;
    });
};

$(document).ready(changePage);

window.setInterval(function(){
    changePage();
}, timer);
