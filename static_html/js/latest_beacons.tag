<latest_beacons>

  <h2>{ opts.title }</h2>

	<div class="progress" id="ws_status">
    <div class="indeterminate blue"></div>
	</div>
  <table>
		<tr>
			<th>Beacon ID</th>
			<th>Beacon Location</th>
			<th>Beacon type</th>
			<th>Last seen</th>
			<th>Distance</th>
			<th>Add</th>
		</tr>
    <tr each={ beacons }>
			<td>{ beacon_id }</td>
			<td>{ beacon_location }</td>
			<td if={beacon_type == ""} >raw</td>
			<td if={beacon_type != ""} >{ beacon_type }</td>
			<td>{ moment(last_seen*1000).fromNow() }</td> 
			<td if={distance != ""} >{ distance }</td>
			<td><a href="#add-beacon/{ beacon_id }">{ add_label }</a></td>
    </tr>
  </table>

	<br />
	<a id="pause_button" class="btn-floating btn-large waves-effect waves-light red"><i class="material-icons">pause</i></a>
	<a id="resume_button" class="btn-floating btn-large waves-effect waves-light red"><i class="material-icons">play_arrow</i></a>

  <script>
    this.beacons = opts.beacons

		var self = this;
		var ws;
		var ws_proto = "ws";
		var check_conn;
		
		var paused = false;

		if (window.location.protocol == "https:") {
			ws_proto = "wss"
		}

		var list_filter = function(data) {
			var beacons = data.map(function(b)
				{
					b.add_label = "Add this beacon";
				if(b.beacon_type=="ibeacon") 
				{
					b.distance = Math.round(parseFloat(b.distance)*100)/100 +" meters";
				}
				else if(b.beacon_type=="eddystone") 
				{
					b.distance = Math.round(parseFloat(b.distance)*100)/100 +" meters";
					if(b.incoming_json.namespace == "ddddeeeeeeffff5544ff")
					{
						b.beacon_type = "HB Button";
					}
				}
				else if(b.beacon_type=="hb_button") 
				{
					b.distance = Math.round(parseFloat(b.distance)*100)/100 +" meters";
					b.add_label = "Add this button";
					b.beacon_type = "HB Button"
				}
				else 
				{
					b.distance = "-"+b.distance+ " db";
				}
				return b;
			});

			return beacons;
		};

		var onmessage = function (evt) 
		{
			if(paused)
			{
				return;
			}

			var data = JSON.parse(evt.data);
							
			data.sort(function(a, b) {
				//return parseInt(a.last_seen) - parseInt(b.last_seen);
				if(a.beacon_id < b.beacon_id) return -1;
				if(a.beacon_id > b.beacon_id) return 1;
				return 0;
			});
			//console.log("latest")
			//console.log(data)
			self.beacons = list_filter(data);
			self.update();
		};
				
		var onopen = function()
		{
			//alert('open show');
			$("#ws_status").removeClass("hide");
		}
    
		var onclose = function()
		{ 
			// websocket is closed.
			$("#ws_status").addClass("hide");
			//alert("Connection is closed...");
		};

		beacon_ws()
		{
			if ("WebSocket" in window)
			{
				ws = new WebSocket(ws_proto+"://"+window.location.host + window.location.pathname+"api/beacons/latest/ws");
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
		})

		checkConnection() {
			if(paused)
			{
				return;
			}
			if(ws.readyState == 3) {
				self.beacon_ws();
			}
		}
	
		refreshList() {
			$.getJSON( "api/latest-beacons", function( data ) {
					//console.log(data)

					data.sort(function(a, b) {
						    //return parseInt(a.last_seen) - parseInt(b.last_seen);
								if(a.beacon_id < b.beacon_id) return -1;
								if(a.beacon_id > b.beacon_id) return 1;
								return 0;
					});
					self.beacons = list_filter(data);
					self.update()
			});
		}

		this.on('mount', function(){
			$("#ws_status").addClass("hide");
			$("#resume_button").hide();

			$("#resume_button").click(function(){
				paused = false;
				self.beacon_ws();
				$("#resume_button").hide();
				$("#pause_button").show();
			});

			$("#pause_button").click(function(){
				paused = true;
				ws.close();
				$("#pause_button").hide();
				$("#resume_button").show();
			});

			self.refreshList();
			self.beacon_ws();
			check_conn = setInterval(this.checkConnection, 1000); 		//do a thing every second
		});
		
	</script>

</latest_beacons>

