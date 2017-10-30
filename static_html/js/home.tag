<home>
  <h2>{ opts.title }</h2>

	<div class="progress" id="ws_status">
    <div class="indeterminate blue"></div>
	</div>
  <table>
		<tr>
			<th>Beacon Name</th>
			<th>Current Location</th>
			<th>Last seen</th>
			<th>Edit</th>
			<th>Delete</th>
		</tr>
    <tr each={ beacons }>
			<td class="{ class }">{ name }</td>
			<td class="{ class }">{ location }</td>
			<td class="{ class }">{ last_seen_string }</td> 
			<td><a href="#edit-beacon/{ beacon_id }/{ url_name }"><i class="material-icons text-blue">edit</i></a></td>
			<td><a onclick={ delete_beacon } beacon_name="{ beacon_name }" beacon_id="{ beacon_id }" href=""><i class="material-icons red-text">delete</i></a></td>
    </tr>
  </table>

	<h2>Added Buttons</h2>

  <table>
		<tr>
			<th>Button Name</th>
			<th>Current Location</th>
			<th>Last seen</th>
			<th>Battery</th>
			<th>Mode</th>
			<th>Edit</th>
			<th>Delete</th>
		</tr>
    <tr each={ buttons }>
			<td class="{ class } tooltipped" data-position="top" data-delay="20" data-tooltip="{ button_id }">{ name }</td> 
			<td class="{ class }">{ button_location }</td>
			<td class="{ class }">{ last_seen_string }</td> 
			<td class="{ class } tooltipped" data-position="top" data-delay="20" data-tooltip="{ hb_button_battery } V">{ hb_button_battery_percent }</td> 
			<td class="{ class }">{ hb_button_mode }</td> 
			<td><a href="#edit-beacon/{ button_id }/{ url_name }"><i class="material-icons text-blue">edit</i></a></td>
			<td><a onclick={ delete_button } beacon_name="{ name }" beacon_id="{ button_id }" href=""><i class="material-icons red-text">delete</i></a></td>
    </tr>
  </table>


	<br />
	<a id="refresh" onclick={ refreshList } class="btn-floating btn-large waves-effect waves-light red"><i class="material-icons">refresh</i></a>

  <script>
    this.beacons = opts.beacons
    this.buttons = opts.buttons

		$('.tooltipped').tooltip({delay: 20});

		var self = this;
		var ws;
		var ws_proto = "ws";
		var check_conn;

		if (window.location.protocol == "https:") {
			ws_proto = "wss"
		}

		var processData = function(data) {
			var bs = []
			var btns = []

			data.buttons.sort(function(a, b) {
					if(a.name < b.name) return -1;
					if(a.name > b.name) return 1;
					return 0;
			});
			data.beacons.sort(function(a, b) {
					if(a.name < b.name) return -1;
					if(a.name > b.name) return 1;
					return 0;
			});

			$.each(data.beacons, function(k, v) {
						if(v) {
							v.last_seen_string = moment(v.last_seen*1000).fromNow() 
							if(v.location == "")
							{
								v.location = "Not Found"
								v.last_seen_string = " - "
								v.class = "grey-text text-darken-1"
							}
							v["url_name"] = encodeURIComponent(v.name);
							bs.push(v);
						}
			});
			$.each(data.buttons, function(k, v) {
						if(v) {
							v.last_seen_string = moment(v.last_seen*1000).fromNow() 
							if(v.location == "")
							{
								v.location = "Not Found"
								v.last_seen_string = " - "
								v.class = "grey-text text-darken-1"
							}
							v["url_name"] = encodeURIComponent(v.name);
							if(v.hb_button_mode == "button_only")
							{
								v.hb_button_mode = "Button Only";
							}
							else 
							{
								v.hb_button_mode = "Beacon & Button";
							}
							if(v.hb_button_battery != "")
							{
								v.hb_button_battery_percent = Math.floor((v.hb_button_battery / 3021)*100)+"%";
								v.hb_button_battery = (v.hb_button_battery / 1000).toFixed(2);
							}
							else 
							{
								v.hb_button_battery_percent = "n/a";
								v.hb_button_battery = "n/a";
							}
							btns.push(v);
						}
			});

			self.beacons = bs;
			self.buttons = btns;
			self.update();
			$('.material-tooltip').remove();
			$('.tooltipped').tooltip({delay: 20});
		};

		//do a thing every second
		refreshList() {
			$.getJSON( "api/results", function( data ) {
				processData(data);	
			});
		}

		var onmessage = function (evt) 
		{
			var data = JSON.parse(evt.data);
			processData(data);	
		};
				
		var onopen = function()
		{
			$("#ws_status").removeClass('hide');
		}
    
		var onclose = function()
		{ 
			// websocket is closed.
			//alert("Connection is closed...");
			$("#ws_status").addClass('hide');
		};

		beacon_ws()
		{
			if ("WebSocket" in window)
			{
				ws = new WebSocket(ws_proto+"://"+window.location.host + window.location.pathname+"api/beacons/ws");
				ws.onopen = onopen;
				ws.onclose = onclose;
				ws.onmessage = onmessage;
			}
            
			else
			{
				// The browser doesn't support WebSocket
				alert("WebSocket NOT supported by your Browser!");
			}
		}

		this.on('unmount', function() {
			ws.close();
			clearInterval(check_conn);
		});
	
		delete_beacon = function() {
				var beacon_id = $(this).attr("beacon_id");
				var beacon_name = $(this).attr("beacon_name");
				var confirmed = confirm("Are you sure you want to delete the '"+beacon_name+"' beacon?");
				if(confirmed !== true)  {
					return false;
				}
				var beacon_id = $(this).attr("beacon_id");
				$.ajax(
				{
					url: "api/beacons/"+beacon_id,
					type: "DELETE",
				})
				.done(function(data) {
					window.location.hash = '#home';
				});

				return false;
		};

	delete_button = function() {
				var beacon_id = $(this).attr("button_id");
				var name = $(this).attr("name");
				var confirmed = confirm("Are you sure you want to delete the '"+name+"' button?");
				if(confirmed !== true)  {
					return false;
				}
				$.ajax(
				{
					url: "api/beacons/"+beacon_id,
					type: "DELETE",
				})
				.done(function(data) {
					window.location.hash = '#home';
				});

				return false;
		};

		this.on('mount', function() {
		});

		checkConnection() {
			if(ws.readyState == 3) {
				$("#ws_status").addClass('hide');
				//is close, try reconnect
				ws = new WebSocket(ws_proto+"://"+window.location.host + window.location.pathname+"/api/beacons/ws");
				ws.onopen = onopen;
				ws.onclose = onclose;
				ws.onmessage = onmessage;
			}
			if(ws.readyState == 1) {
				$("#ws_status").removeClass('hide');
			}
		}

		$("#ws_status").addClass('hide');
		self.refreshList();
		self.beacon_ws();
		check_conn = setInterval(this.checkConnection, 1000); 

	</script>

</home>

