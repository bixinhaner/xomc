<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>
<%@ include file="/common/loading.jsp" %>

<style>
	.apnNum_row {
		height: 40px;
		margin-top: 15px;
	}
	.cpeApnTable_col {
		display: flex;
		flex-direction: column;
		justify-content: space-between;
	}
	#cpeApnTable .inputTipCss {
		font-size: 10px !important;
	}
	.title_row {
		color: #a6a9b3;
		font-weight: bold;
	}
</style>
<!--  <div style="padding:25px 50x 5px;margin-left:4%;margin-top:3%;position:absolute;top:0px;left:0px;right:0px;bottom:0px">
	<div id="cpeSettingTittle"  style="display:inline-block;padding:0 10px;color:#7993B6;font-size:16px;">APN</div>
	<div class="tableDiv titleIcon_close" style="position:absolute;right:5%;" onclick="closeApnSet()"></div>
</div>  -->
<div class="slidebarTitleDiv">
	<div style="display: inline-block;">APN</div>
</div>
<!--<div class="circleIcon placeholder-bt" placeholder="<%=rb.getString("GuanBi")%>" onclick="closeApnSet()">
	<span class="el-icon el-icon-circle-close"></span>
</div>-->
<div class="flex-ctn" style="padding:0 15px 15px;overflow-y:auto;position:absolute;top:82px;left:40px;right:0px;bottom:10px;margin-top:0px;">
	<div class="flex-item">
      <!--  apn设置 -->
       	<div id="cpeApnDiv" style="margin-bottom:40px;">
			<!--<div class="group-title not-extend">
				<span class="title-icon"></span>
				<span class="title-text">CPE APN/L2 Setting</span>
			</div>-->
			<div id="cpeApnDetails" style="margin-top:5px;">
				<div id="l2ServerIp" class="qosOptionDetailsLeft" >
					<label class="inputTittleCss">L2 Server IP:</label>
					<input class="inputDivCss border border-box" oldValue=""  id="vxlanserverip" name="l2ServerIp_NAME" onfocus="ipAddressChangeRed(this)" value ="${vxlanserverip}"/>
					<label class="inputTipCss">IP Address</label>
				</div>
				<div id="cpeApnTable" style="margin-top:20px;display:flex;justify-content:space-between;min-width:1000px;">
					<div class="apnNum_col cpeApnTable_col" style="width:30px">
						<div class="title_row"> . </div>
						<div class="apnNum1_row apnNum_row ">APN1</div>
						<div class="apnNum2_row apnNum_row ">APN2</div>
						<div class="apnNum3_row apnNum_row ">APN3</div>
						<div class="apnNum4_row apnNum_row ">APN4</div>
					</div>
					<div class="enable_col cpeApnTable_col" style="width:40px">
						<div class="title_row">Enable</div>
						<div class="apnNum1_row apnNum_row"><input type="checkbox" id="apnenable1"  value="0" oldValue="0" onclick="apnCheckFun(this)"/></div>
						<div class="apnNum2_row apnNum_row"><input type="checkbox" id="apnenable2"  value="0" oldValue="0" onclick="apnCheckFun(this)"/></div>
						<div class="apnNum3_row apnNum_row"><input type="checkbox" id="apnenable3" 	value="0" oldValue="0" onclick="apnCheckFun(this)"/></div>
						<div class="apnNum4_row apnNum_row"><input type="checkbox" id="apnenable4" 	value="0" oldValue="0" onclick="apnCheckFun(this)"/></div>
					</div>
					<div class="apnType_col cpeApnTable_col" style="width:60px">
						<div class="title_row">APN Type</div>
						<div class="apnNum1_row apnNum_row apnType_row">
							<select class="border-box border" name="" id="apntype1"  oldValue="001" style="height:26px;width:60px;" onchange="apnSelectFun(this)">
								<option value="001" selected="selected">L2</option>
								<option value="000">L3</option>
							</select>
						</div>
						<div class="apnNum2_row apnNum_row apnType_row">
							<select class="border-box border" name="" id="apntype2"  oldValue="001" style="height:26px;width:60px;" onchange="apnSelectFun(this)">
								<option value="001" selected="selected">L2</option>
								<option value="000">L3</option>
							</select>
						</div>
						<div class="apnNum3_row apnNum_row apnType_row">
							<select class="border-box border" name="" id="apntype3"  oldValue="001" style="height:26px;width:60px;" onchange="apnSelectFun(this)">
								<option value="001" selected="selected">L2</option>
								<option value="000">L3</option>
							</select>
						</div>
						<div class="apnNum4_row apnNum_row apnType_row">
							<select class="border-box border" name="" id="apntype4"   oldValue="001" style="height:26px;width:60px;" onchange="apnSelectFun(this)">
								<option value="001" selected="selected">L2</option>
								<option value="000">L3</option>
							</select>
						</div>
					</div>
					<div class="apnName_col cpeApnTable_col" style="width:200px;">
						<div class="title_row">APN Name</div>
						<div class="apnNum1_row apnNum_row apnName_row">								
							<input class="border border-box" style="height: 26px;width:200px;" onfocus="apNameChangeRed(this)" id="apnname1" oldValue=""/>
							<label class="inputTipCss"><%=rb.getString("ApnNameZuChengTiShi")%></label>
						</div>
						<div class="apnNum2_row apnNum_row apnName_row">								
							<input class="border border-box" style="height: 26px;width:200px;" onfocus="apNameChangeRed(this)" id="apnname2" oldValue=""/>
							<label class="inputTipCss"><%=rb.getString("ApnNameZuChengTiShi")%></label>
						</div>
						<div class="apnNum3_row apnNum_row apnName_row">								
							<input class="border border-box" style="height: 26px;width:200px;" onfocus="apNameChangeRed(this)" id="apnname3" oldValue=""/>
							<label class="inputTipCss"><%=rb.getString("ApnNameZuChengTiShi")%></label>
						</div>
						<div class="apnNum4_row apnNum_row apnName_row">								
							<input class="border border-box" style="height: 26px;width:200px;" onfocus="apNameChangeRed(this)" id="apnname4" oldValue=""/>
							<label class="inputTipCss"><%=rb.getString("ApnNameZuChengTiShi")%></label>
						</div>
					</div>
					<div class="defaultRouter_col cpeApnTable_col" style="width:86px">
						<div class="title_row">Default Router</div>
						<div class="apnNum1_row apnNum_row defaultRouter_row"><input type ="checkbox" id="defaultrouter1" value="0" oldValue="0"/></div>
						<div class="apnNum2_row apnNum_row defaultRouter_row"><input type ="checkbox" id="defaultrouter2" value="0" oldValue="0"/></div>
						<div class="apnNum3_row apnNum_row defaultRouter_row"><input type ="checkbox" id="defaultrouter3" value="0" oldValue="0"/></div>
						<div class="apnNum4_row apnNum_row defaultRouter_row"><input type ="checkbox" id="defaultrouter4" value="0" oldValue="0"/></div>
					</div>
					<div class="vlanList_col cpeApnTable_col" style="width:240px">
						<div class="title_row">Vlan List</div>
						<div class="apnNum1_row apnNum_row vlanList_row">								
							<input class="border border-box" style="height: 26px;width:230px;" onfocus="vlanListChangeRed(this)" id="vlan1"  oldValue=""/>
							<label class="inputTipCss"><%=rb.getString("VlanListTiShi")%></label>
						</div>
						<div class="apnNum2_row apnNum_row vlanList_row">								
							<input class="border border-box" style="height: 26px;width:230px;" onfocus="vlanListChangeRed(this)" id="vlan2"  oldValue="" />
							<label class="inputTipCss"><%=rb.getString("VlanListTiShi")%></label>
						</div>
						<div class="apnNum3_row apnNum_row vlanList_row">								
							<input class="border border-box" style="height: 26px;width:230px;" onfocus="vlanListChangeRed(this)" id="vlan3"  oldValue="" />
							<label class="inputTipCss"><%=rb.getString("VlanListTiShi")%></label>
						</div>
						<div class="apnNum4_row apnNum_row vlanList_row">								
							<input class="border border-box" style="height: 26px;width:230px;" onfocus="vlanListChangeRed(this)" id="vlan4"  oldValue="" />
							<label class="inputTipCss"><%=rb.getString("VlanListTiShi")%></label>
						</div>
					</div>
					<div class="apnIp_col cpeApnTable_col" style="width:130px">
						<div class="title_row">IP</div>
						<div class="apnNum1_row apnNum_row">								
							<input class="border border-box" style="height: 26px;width:130px;" disabled="disabled" id="apnip1"  oldValue="" />
						</div>
						<div class="apnNum2_row apnNum_row">								
							<input class="border border-box" style="height: 26px;width:130px;" disabled="disabled" id="apnip2"  oldValue="" />
						</div>
						<div class="apnNum3_row apnNum_row">								
							<input class="border border-box" style="height: 26px;width:130px;" disabled="disabled" id="apnip3"   oldValue="" />
						</div>
						<div class="apnNum4_row apnNum_row">								
							<input class="border border-box" style="height: 26px;width:130px;" disabled="disabled" id="apnip4"   oldValue=""/>
						</div>
					</div>
				</div>
			</div>
		</div> 
	</div>
	<div id="operateDiv" style="width:91%;margin-left: 3%;">
		<div id="operateDivDetails">
			<div class="windowButtonGroup" style="float:left;">					
				<a class="linkbutton linkbutton_trend" href="javascript:void(0)" onclick="apnCommit()"><span><%=rb.getString("QueDing")%></span></a>
				<a class="linkbutton linkbutton_nowanna" href="javascript:void(0)" onclick="closeApnSet()"><span><%=rb.getString("QuXiao")%></span></a>
			</div>
		</div>
	</div>	
</div>
<script>
	var cpeCode = "${cpeCode}";
	$(function(){
		closeLoading();
		//初始化默认未勾选，所有都不可操作
		$(".enable_col input").each(function(){
			var operateClassName = $(this).parent()[0].className.split(" ")[0];
			if(!$(this).is(":checked")){
				$("."+operateClassName).each(function(){
		            $($(this)[0]).children(":first-child").prop("disabled","disabled");
		        })
		        $(this).prop("disabled","");
			}else{
				$("."+operateClassName).each(function(){
		            $($(this)[0]).children(":first-child").prop("disabled","");
		        })
			}
		}) 
		$(".apnIp_col input").prop("disabled","disabled");
		$("#vxlanserverip").val("");
		$("#vid").val("");
		$("#apnname1").val("");
		$("#vlan1").val("");
		$("#apnname2").val("");
		$("#vlan2").val("");
		$("#apnname3").val("");
		$("#vlan3").val("");
		$("#apnname4").val("");
		$("#vlan4").val("");
		$("#apnip1").val("");
		$("#apnip2").val("");
		$("#apnip3").val("");
		$("#apnip4").val("");
		$("#vxlanserverip").attr("oldValue","");
		$("#vid").attr("oldValue","");
		$("#apnname1").attr("oldValue","");
		$("#apnname2").attr("oldValue","");
		$("#apnname3").attr("oldValue","");
		$("#apnname4").attr("oldValue","");
		
		$("#vlan1").attr("oldValue","");
		$("#vlan2").attr("oldValue","");
		$("#vlan3").attr("oldValue","");
		$("#vlan4").attr("oldValue","");
		
		$("#defaultrouter1").attr("oldValue","");
		$("#defaultrouter2").attr("oldValue","");
		$("#defaultrouter3").attr("oldValue","");
		$("#defaultrouter4").attr("oldValue","");
		
		$("#apnenable1").attr("oldValue","");
		$("#apnenable2").attr("oldValue","");
		$("#apnenable3").attr("oldValue","");
		$("#apnenable4").attr("oldValue","");
		
		
		$("#apntype1").attr("oldValue","");
		$("#apntype2").attr("oldValue","");
		$("#apntype3").attr("oldValue","");
		$("#apntype4").attr("oldValue","");
	 	$.ajax({
			type: "post",
			url: "${ctx}/cell/CPE/getCPEL2Infos.action", 
			data: {"cpeCode": cpeCode},
			async: false,
			dataType: 'json',
			success: function(data) {
				if("0"== data["connFlag"]){
					 $("#vxlanserverip").prop("disabled","disabled");
				} else {
					 $("#vxlanserverip").prop("disabled","");
				}
				
				var conn = data["connFlag"];
				var aname1 = data["apnname1"];
				var aname2 = data["apnname2"];
				var aname3 = data["apnname3"];
				var aname4 = data["apnname4"];
		
				var enable1 = data["enable1"];
				var enable2 = data["enable2"];
				var enable3 = data["enable3"];
				var enable4 = data["enable4"];
				
				var apntype1 = data["apntype1"];
				var apntype2 = data["apntype2"];
				var apntype3 = data["apntype3"];
				var apntype4 = data["apntype4"];
				
				
				var defaultRouter1 = data["defaultRouter1"];
				var defaultRouter2 = data["defaultRouter2"];
				var defaultRouter3 = data["defaultRouter3"];
				var defaultRouter4 = data["defaultRouter4"];
				
				
				var ip1 = data["ip1"];
				var ip2 = data["ip2"];
				var ip3 = data["ip3"];
				var ip4 = data["ip4"];
				
				
				if(enable1){
					$("#apnenable1").attr("oldValue",enable1);
					$("#apnenable1").attr("value",enable1);
					if(enable1 == "1"){
						$("#apnenable1").prop("checked",true);
						$("#apntype1").prop("disabled","");
						$("#apnname1").prop("disabled","");
						$("#defaultrouter1").prop("disabled","");
						$("#vlan1").prop("disabled","");
					} else {
						$("#apnenable1").prop("checked",false);
						$("#apntype1").prop("disabled","disabled");
						$("#apnname1").prop("disabled","disabled");
						$("#defaultrouter1").prop("disabled","disabled");
						$("#vlan1").prop("disabled","disabled");
					}
					$("#apnenable1").prop("disabled","");
				} else {
					$("#apnenable1").attr("oldValue","0");
					$("#apnenable1").attr("value","0");
					$("#apnenable1").prop("disabled","");
					$("#apnenable1").prop("checked",false);
					$("#apntype1").prop("disabled","disabled");
					$("#apnname1").prop("disabled","disabled");
					$("#defaultrouter1").prop("disabled","disabled");
					$("#vlan1").prop("disabled","disabled");
				}
				if(enable2){
					$("#apnenable2").attr("oldValue",enable2);
					$("#apnenable2").attr("value",enable2);
					if(enable2 == "1"){
						$("#apnenable2").prop("checked",true);
						$("#apntype2").prop("disabled","");
						$("#apnname2").prop("disabled","");
						$("#defaultrouter2").prop("disabled","");
						$("#vlan2").prop("disabled","");
					} else {
						$("#apnenable2").prop("checked",false);
						$("#apntype2").prop("disabled","disabled");
						$("#defaultrouter2").prop("disabled","disabled");
						$("#apnname2").prop("disabled","disabled");
						$("#vlan2").prop("disabled","disabled");
					}
					$("#apnenable2").prop("disabled","");
				} else {
					$("#apnenable2").attr("oldValue","0");
					$("#apnenable2").attr("value","0");
					$("#apnenable2").prop("disabled","");
					$("#apnenable2").prop("checked",false);
					$("#apntype2").prop("disabled","disabled");
					$("#apnname2").prop("disabled","disabled");
					$("#defaultrouter2").prop("disabled","disabled");
					$("#vlan2").prop("disabled","disabled");
				}
				
				if(enable3){
					$("#apnenable3").attr("oldValue",enable3);
					$("#apnenable3").attr("value",enable3);
					if(enable3 == "1"){
						$("#apnenable3").prop("checked",true);
						$("#apntype3").prop("disabled","");
						$("#apnname3").prop("disabled","");
						$("#defaultrouter3").prop("disabled","");
						$("#vlan3").prop("disabled","");
					} else {
						$("#apnenable3").prop("checked",false);
						$("#apntype3").prop("disabled","disabled");
						$("#apnname3").prop("disabled","disabled");
						$("#defaultrouter3").prop("disabled","disabled");
						$("#vlan3").prop("disabled","disabled");
					}
					$("#apnenable3").prop("disabled","");
				} else {
					$("#apnenable3").attr("oldValue","0");
					$("#apnenable3").attr("value","0");
					$("#apnenable3").prop("disabled","");
					$("#apnenable3").prop("checked",false);
					$("#apntype3").prop("disabled","disabled");
					$("#apnname3").prop("disabled","disabled");
					$("#defaultrouter3").prop("disabled","disabled");
					$("#vlan3").prop("disabled","disabled");
				}
				
				if(enable4){
					$("#apnenable4").attr("oldValue",enable4);
					$("#apnenable4").attr("value",enable4);
					if(enable4 == "1"){
						$("#apnenable4").prop("checked",true);
						$("#apntype4").prop("disabled","");
						$("#apnname4").prop("disabled","");
						$("#defaultrouter4").prop("disabled","");
						$("#vlan4").prop("disabled","");
					} else {
						$("#apnenable4").prop("checked",false);
						$("#apntype4").prop("disabled","disabled");
						$("#apnname4").prop("disabled","disabled");
						$("#defaultrouter4").prop("disabled","disabled");
						$("#vlan4").prop("disabled","disabled");
					}
					$("#apnenable4").prop("disabled","");
				} else {
					$("#apnenable4").attr("oldValue","0");
					$("#apnenable4").attr("value","0");
					$("#apnenable4").prop("disabled","");
					$("#apnenable4").prop("checked",false);
					$("#apntype4").prop("disabled","disabled");
					$("#apnname4").prop("disabled","disabled");
					$("#defaultrouter4").prop("disabled","disabled");
					$("#vlan4").prop("disabled","disabled");
				}
				
				if(apntype1){
					$("#apntype1").attr("oldValue",apntype1);
					if(apntype1 == "001"){//L2
						$("#apntype1").val(apntype1);
						$("#defaultrouter1").prop("disabled","disabled");
					} else if(apntype1 == "000"){//L3
						$("#vlan1").prop("disabled","disabled");
						$("#apntype1").val(apntype1);
					}
					//$("#apntype1").prop("disabled","");
				} else {
					$("#apntype1").attr("oldValue","001");
				}
				
				if(apntype2){
					$("#apntype2").attr("oldValue",apntype2);
					if(apntype2 == "001"){
						$("#apntype2").val(apntype2);
						$("#defaultrouter2").prop("disabled","disabled");
					} else if(apntype2 == "000"){
						$("#vlan2").prop("disabled","disabled");
						$("#apntype2").val(apntype2);
					}
					//$("#apntype2").prop("disabled","");
				} else {
					$("#apntype2").attr("oldValue","001");
				}
				
				if(apntype3){
					$("#apntype3").attr("oldValue",apntype3);
					if(apntype3 == "001"){
						$("#defaultrouter3").prop("disabled","disabled");
						$("#apntype3").val(apntype3);
					} else if(apntype3 == "000"){
						$("#vlan3").prop("disabled","disabled");
						$("#apntype3").val(apntype3);
					}
					//$("#apntype3").prop("disabled","");
				} else {
					$("#apntype3").attr("oldValue","001");
				}
				
				if(apntype4){
					$("#apntype4").attr("oldValue",apntype4);
					if(apntype4 == "001"){
						$("#defaultrouter4").prop("disabled","disabled");
						$("#apntype4").val(apntype4);
					} else if(apntype4 == "000"){
						$("#vlan4").prop("disabled","disabled");
						$("#apntype4").val(apntype4);
					}
					//$("#apntype4").prop("disabled","");
				} else {
					$("#apntype4").attr("oldValue","001");
				}
				
				if(defaultRouter1){
					$("#defaultrouter1").attr("oldValue",defaultRouter1);
					$("#defaultrouter1").attr("value",defaultRouter1);
					if(defaultRouter1 == "1"){
						$("#defaultrouter1").prop("checked",true);
					} else if(defaultRouter1 == "0"){
						$("#defaultrouter1").prop("checked",false);
					}
					if(enable1 == "0"){
						$("#defaultrouter1").prop("disabled","disabled");
					}else{
						$("#defaultrouter1").prop("disabled","");
					}
				} else {
					$("#defaultrouter1").attr("oldValue","0");
					$("#defaultrouter1").attr("value","0");
				}
				
				if(defaultRouter2){
					$("#defaultrouter2").attr("oldValue",defaultRouter2);
					$("#defaultrouter2").attr("value",defaultRouter2);
					if(defaultRouter2 == "1"){
						$("#defaultrouter2").prop("checked",true);
					} else if(defaultRouter2 == "0"){
						$("#defaultrouter2").prop("checked",false);
					}
					if(enable2 == "0"){
						$("#defaultrouter2").prop("disabled","disabled");
					}else{
						$("#defaultrouter2").prop("disabled","");
					}
				} else {
					$("#defaultrouter2").attr("oldValue","0");
					$("#defaultrouter2").attr("value","0");
				}
				
				if(defaultRouter3){
					$("#defaultrouter3").attr("oldValue",defaultRouter3);
					$("#defaultrouter3").attr("value",defaultRouter3);
					if(defaultRouter3 == "1"){
						$("#defaultrouter3").prop("checked",true);
					} else if(defaultRouter3 == "0"){
						$("#defaultrouter3").prop("checked",false);
					}
					if(enable3 == "0"){
						$("#defaultrouter3").prop("disabled","disabled");
					}else{
						$("#defaultrouter3").prop("disabled","");
					}
				} else {
					$("#defaultrouter3").attr("oldValue","0");
					$("#defaultrouter3").attr("value","0");
				}
				
				if(defaultRouter4){
					$("#defaultrouter4").attr("oldValue",defaultRouter4);
					$("#defaultrouter4").attr("value",defaultRouter4);
					if(defaultRouter4 == "1"){
						$("#defaultrouter4").prop("checked",true);
					} else if(defaultRouter4 == "0"){
						$("#defaultrouter4").prop("checked",false);
					}
					if(enable4 =="0"){
						$("#defaultrouter4").prop("disabled","disabled");
					}else{
						$("#defaultrouter4").prop("disabled","");
					}
				} else {
					$("#defaultrouter4").attr("oldValue","0");
					$("#defaultrouter4").attr("value","0");
				}
				if(ip1){
					$("#apnip1").attr("oldValue",ip1);
					$("#apnip1").val(ip1);
					$("#apnip1").prop("disabled","disabled");
				}
				if(ip2){
					$("#apnip2").attr("oldValue",ip2);
					$("#apnip2").val(ip2);
					$("#apnip2").prop("disabled","disabled");
				}
				if(ip3){
					$("#apnip3").attr("oldValue",ip3);
					$("#apnip3").val(ip3);
					$("#apnip3").prop("disabled","disabled");
				}
				if(ip4){
					$("#apnip4").attr("oldValue",ip4);
					$("#apnip4").val(ip4);
					$("#apnip4").prop("disabled","disabled");
				}
				
				var vxlanserver = data["vxlanserver"];
				var vid = data["vni"];
				
				$("#vxlanserverip").val(vxlanserver);
				$("#vid").val(vid);
				if(aname1){
					$("#apnname1").val(aname1.split("-")[0]);
					$("#vlan1").val(aname1.split("-")[1].replace(/\s/g,","));
				}
				if(aname2){
					$("#apnname2").val(aname2.split("-")[0]);
					$("#vlan2").val(aname2.split("-")[1].replace(/\s/g,","));
				}
				if(aname3){
					$("#apnname3").val(aname3.split("-")[0]);
					$("#vlan3").val(aname3.split("-")[1].replace(/\s/g,","));
				}
				if(aname4){
					$("#apnname4").val(aname4.split("-")[0]);
					$("#vlan4").val(aname4.split("-")[1].replace(/\s/g,","));
				}
				
			 $("#vxlanserverip").attr("oldValue",vxlanserver);
			 $("#vid").attr("oldValue",vid);
			 if(aname1){
				 $("#apnname1").attr("oldValue",aname1.split("-")[0]);
				 $("#vlan1").attr("oldValue",aname1.split("-")[1].replace(/\s/g,","));
			 }
			 if(aname2){
				 $("#apnname2").attr("oldValue",aname2.split("-")[0]);
				 $("#vlan2").attr("oldValue",aname2.split("-")[1].replace(/\s/g,","));
			 }
			 if(aname3){
				 $("#apnname3").attr("oldValue",aname3.split("-")[0]);
				 $("#vlan3").attr("oldValue",aname3.split("-")[1].replace(/\s/g,","));
			 }
			 if(aname4){
				 $("#apnname4").attr("oldValue",aname4.split("-")[0]);
				 $("#vlan4").attr("oldValue",aname4.split("-")[1].replace(/\s/g,","));
			 }
			 
			 if(conn == "0"){
				 disabledall();
			 }
				
			}
		});
	})
	function disabledall(){
		$("#vxlanserverip").prop("disabled","disabled");
		$("#cpeApnDetails .enable_col input").each(function(){
			var operateClassName = $(this).parent()[0].className.split(" ")[0];
			$("."+operateClassName).each(function(){
	            $($(this)[0]).children(":first-child").prop("disabled","disabled");
	        })
		})
	}
	function validateQoSIsChange(){
		var vschangeflag = true;
		var av1changeflag = true;
		var av2changeflag = true;
		var av3changeflag = true;
		var av4changeflag = true;
		
		var serverIPoldValue = $("#vxlanserverip").attr("oldValue");
		var idoldValue = $("#vid").attr("oldValue");
		var apnname1oldValue = $("#apnname1").attr("oldValue");
		var apnname2oldValue = $("#apnname2").attr("oldValue");
		var apnname3oldValue = $("#apnname3").attr("oldValue");
		var apnname4oldValue = $("#apnname4").attr("oldValue");
		var vlan1oldValue = $("#vlan1").attr("oldValue");
		var vlan2oldValue = $("#vlan2").attr("oldValue");
		var vlan3oldValue = $("#vlan3").attr("oldValue");
		var vlan4oldValue = $("#vlan4").attr("oldValue");
		
		var defaultrouter1oldValue = $("#defaultrouter1").attr("oldValue");
		var defaultrouter2oldValue = $("#defaultrouter2").attr("oldValue");
		var defaultrouter3oldValue = $("#defaultrouter3").attr("oldValue");
		var defaultrouter4oldValue = $("#defaultrouter4").attr("oldValue");
		
		var enable1oldValue = $("#apnenable1").attr("oldValue");
		var enable2oldValue = $("#apnenable2").attr("oldValue");
		var enable3oldValue = $("#apnenable3").attr("oldValue");
		var enable4oldValue = $("#apnenable4").attr("oldValue");
		
		
		var apntype1oldValue = $("#apntype1").attr("oldValue");
		var apntype2oldValue = $("#apntype2").attr("oldValue");
		var apntype3oldValue = $("#apntype3").attr("oldValue");
		var apntype4oldValue = $("#apntype4").attr("oldValue");
		
		
		var ip1oldValue = $("#ip1").attr("oldValue");
		var ip2oldValue = $("#ip2").attr("oldValue");
		var ip3oldValue = $("#ip3").attr("oldValue");
		var ip4oldValue = $("#ip4").attr("oldValue");

		var id = $("#vid").val();
		var serverIP = $("#vxlanserverip").val().replace(/\s/g,"");
		
		if(id == idoldValue && serverIPoldValue == serverIP){
			vschangeflag = false;
		}
		//获取apnname vlan并校验
		var apnname1 = $("#apnname1").val();
		var vlan1 = $("#vlan1").val();
		var enable1 = $("#apnenable1").get(0).checked ? "1" : "0";
		var df1 = $("#defaultrouter1").get(0).checked ? "1" : "0";
		var apntype1 = $("#apntype1").val();
		if(vlan1 == vlan1oldValue  &&  apnname1oldValue == apnname1 && enable1oldValue ==enable1 && apntype1oldValue == apntype1 && defaultrouter1oldValue == df1){
			av1changeflag = false;
		}
		
		var apnname2 = $("#apnname2").val();
		var vlan2 = $("#vlan2").val();
		var enable2 = $("#apnenable2").get(0).checked ? "1" : "0";
		var df2 = $("#defaultrouter2").get(0).checked ? "1" : "0";
		var apntype2 = $("#apntype2").val();
		if(vlan2 == vlan2oldValue  &&  apnname2oldValue == apnname2 && enable2oldValue ==enable2 && apntype2oldValue == apntype2 && defaultrouter2oldValue == df2){
			av2changeflag = false;
		}
		
		var apnname3 = $("#apnname3").val();
		var vlan3 = $("#vlan3").val();
		var enable3 = $("#apnenable3").get(0).checked ? "1" : "0";
		var df3 = $("#defaultrouter3").get(0).checked ? "1" : "0";
		var apntype3 = $("#apntype3").val();
			if(vlan3 == vlan3oldValue  &&  apnname3oldValue == apnname3 && enable3oldValue ==enable3 && apntype3oldValue == apntype3 && defaultrouter3oldValue == df3){
			av3changeflag = false;
		}
			
		var apnname4 = $("#apnname4").val();
		var vlan4 = $("#vlan4").val();
		var enable4 = $("#apnenable4").get(0).checked ? "1" : "0";
		var df4 = $("#defaultrouter4").get(0).checked ? "1" : "0";
		var apntype4 = $("#apntype4").val();	
		if(vlan4 == vlan4oldValue  &&  apnname4oldValue == apnname4 && enable4oldValue ==enable4 && apntype4oldValue == apntype4 && defaultrouter4oldValue == df4){
			av4changeflag = false;
		}
		
		if(!(vschangeflag || av1changeflag || av2changeflag || av3changeflag || av4changeflag)){
			return false;
		} else{
			return true;
		}	
	}
	/* 验证id是否有效 */
	function validateId(id) {
	    var reg = /^(0|\d{1,24})$/;
	    if (reg.test(id)) {
	        return true;
	    } else {
	        return false;
	    }
	}
	/* 验证是否为合法的IP地址 */
	function validateVxlanIPAddress(ip) {
		var reg = /^(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])$/;
		if (!reg.test(ip)) {
			return false;
		} 
		return true;
	}
	function validateApnname(apnname){
		 var reg = /^([a-zA-Z_])+([a-zA-Z0-9_])*$/;
		 return reg.test(apnname);
	}
	function validateVlan(vlan){
		var reg = /^(\d+,?)+$/;
		if(!reg.test(vlan)){
			return false;
		}
		var arrVlan = vlan.split(",");
		for(var i in arrVlan){
			if((arrVlan[i] < 4 || arrVlan[i] > 4094) ){
				return false;
			}
		}
		return true;
	}
	function setQos(){	
		var serverIPoldValue = $("#vxlanserverip").attr("oldValue");
		var idoldValue = $("#vid").attr("oldValue");
		var apnname1oldValue = $("#apnname1").attr("oldValue");
		var apnname2oldValue = $("#apnname2").attr("oldValue");
		var apnname3oldValue = $("#apnname3").attr("oldValue");
		var apnname4oldValue = $("#apnname4").attr("oldValue");
		var vlan1oldValue = $("#vlan1").attr("oldValue");
		var vlan2oldValue = $("#vlan2").attr("oldValue");
		var vlan3oldValue = $("#vlan3").attr("oldValue");
		var vlan4oldValue = $("#vlan4").attr("oldValue");
		
		var defaultrouter1oldValue = $("#defaultrouter1").attr("oldValue");
		var defaultrouter2oldValue = $("#defaultrouter2").attr("oldValue");
		var defaultrouter3oldValue = $("#defaultrouter3").attr("oldValue");
		var defaultrouter4oldValue = $("#defaultrouter4").attr("oldValue");
		
		var enable1oldValue = $("#apnenable1").attr("oldValue");
		var enable2oldValue = $("#apnenable2").attr("oldValue");
		var enable3oldValue = $("#apnenable3").attr("oldValue");
		var enable4oldValue = $("#apnenable4").attr("oldValue");
		
		
		var apntype1oldValue = $("#apntype1").attr("oldValue");
		var apntype2oldValue = $("#apntype2").attr("oldValue");
		var apntype3oldValue = $("#apntype3").attr("oldValue");
		var apntype4oldValue = $("#apntype4").attr("oldValue");
		
		
		var ip1oldValue = $("#ip1").attr("oldValue");
		var ip2oldValue = $("#ip2").attr("oldValue");
		var ip3oldValue = $("#ip3").attr("oldValue");
		var ip4oldValue = $("#ip4").attr("oldValue");
		
		
		var params = {};
		var paramVlan = "";
		var paramVlanHeader = ""; 
		var vlanList = "";
		//id & vxlan server ip
		var id = 0;
		var serverIP = $("#vxlanserverip").val().replace(/\s/g,"");
		if(!validateVxlanIPAddress(serverIP)){
			if(serverIP.length<1){
				showMsg('prompt_msg','<%=rb.getString("QingShuRuFuWuQiIP")%>');
				alertFlag = true;
				return;
			}
			
		}
		if(!validateId(id)){
			if(id.length<1){
				alertFlag = true;
				showMsg('prompt_msg','<%=rb.getString("QingShuRuId")%>');
				return;
			}
			alertFlag = true;
			showMsg('prompt_msg','<%=rb.getString("VNIChaoChuFanWei")%>');
			return;
		}
		
		paramVlanHeader = paramVlanHeader + "vxlanserver:" + serverIP +"," + "vni:" + id + "," +"enable:1,";
		
		paramVlan = paramVlan +"apnmap:"
		//获取apnname vlan并校验
		var apnname1 = $("#apnname1").val();
		var vlan1 = $("#vlan1").val();
		var enable1 = $("#apnenable1").get(0).checked ? "1" : "0";
		var apntype1 = $("#apntype1").val();
		var df1 = $("#defaultrouter1").get(0).checked ? "1" : "0";
		var ip1 = $("#apnip1").val();
		if(!ip1){
			ip1="";
		}
		var L3Num = 0;
		var L3dfNum = 0;
		if(enable1 == "1"){
			var apnFlag = validateApnname(apnname1);
			var vlanFlag = validateVlan(vlan1);
			if(apntype1 == "000"){//L3 只需验证APNName + defaultrouter
				L3Num = L3Num + 1;//记录L3个数
				if(apnname1){
					if(!apnFlag){
						alertFlag = true;
						showMsg('prompt_msg','<%=rb.getString("APN1MingChengGeShiBuZhengQue")%>');
						return;
					}
					if(df1 == "1"){//df选中
						L3dfNum = L3dfNum + 1;
					}
					var flag = apntype1 + enable1 + df1;
					paramVlan = paramVlan  + "(" + apnname1 + "|" + flag + "|" + ip1 + "|" + ")";
					
				} else {
					alertFlag = true;
					showMsg('prompt_msg','<%=rb.getString("QingShuRuApnName1")%>');
					return;
				}
			} else {//L2
				if(apnname1 && vlan1){
					//都存在输入，则校验
					var apnFlag = validateApnname(apnname1);
					var vlanFlag = validateVlan(vlan1);
					if(!apnFlag){
						alertFlag = true;
						showMsg('prompt_msg','<%=rb.getString("APN1MingChengGeShiBuZhengQue")%>');
						return;
					}
					if(!vlanFlag){
						alertFlag = true;
						showMsg('prompt_msg','<%=rb.getString("VLAN1MingChengGeShiBuZhengQue")%>');
						return;
					}
					//都通过，则输入正确
					var flag = apntype1 + enable1 + df1;
					paramVlan = paramVlan  + "(" + apnname1 + "|"+ flag + "|" + ip1 + vlan1.replace(/,/g," ") + ")";
					vlanList = vlan1;
					
				}else {
					if(!apnname1 && (vlan1.length > 0) ){
						alertFlag = true;
						showMsg('prompt_msg','<%=rb.getString("QingShuRuApnName1")%>');
						return;
					} 
					if(!vlan1 && (apnname1.length > 0) ){
						alertFlag = true;
						showMsg('prompt_msg','<%=rb.getString("QingShuRuVlanList1")%>');
						return;
					}
					paramVlan = paramVlan + "()";
				}
				
			}
		} else {
			paramVlan = paramVlan + "()";
			alertFlag = true;
			showMsg('prompt_msg',"<%=rb.getString("DiYiGeEnableBiXuan")%>");
			return;
		}
		
		var apnname2 = $("#apnname2").val();
		var vlan2 = $("#vlan2").val();
		var enable2 = $("#apnenable2").get(0).checked ? "1" : "0";
		var apntype2 = $("#apntype2").val();
		var df2 = $("#defaultrouter2").get(0).checked ? "1" : "0";
		var ip2 = $("#apnip2").val();
		if(!ip2){
			ip2="";
		}
		if(enable2 == "1"){
			var apnFlag = validateApnname(apnname2);
			var vlanFlag = validateVlan(vlan2);
			if(apntype2 == "000"){//L3 只需验证APNName + defaultrouter
				L3Num = L3Num + 1;//记录L3个数
				if(apnname2){
					if(!apnFlag){
						alertFlag = true;
						showMsg('prompt_msg','<%=rb.getString("APN2MingChengGeShiBuZhengQue")%>');
						return;
					}
					if(df2 == "1"){//df选中
						L3dfNum = L3dfNum + 1;
					}
					var flag = apntype2 + enable2 + df2;
					paramVlan = paramVlan  + "(" + apnname2 + "|" + flag + "|" + ip2 + "|" + ")";
					
				} else {
					alertFlag = true;
					showMsg('prompt_msg','<%=rb.getString("QingShuRuApnName2")%>');
					return;
				}
			} else {//L2
				if(apnname2 && vlan2){
					//都存在输入，则校验
					var apnFlag = validateApnname(apnname2);
					var vlanFlag = validateVlan(vlan2);
					if(!apnFlag){
						alertFlag = true;
						showMsg('prompt_msg','<%=rb.getString("APN2MingChengGeShiBuZhengQue")%>');
						return;
					}
					if(!vlanFlag){
						alertFlag = true;
						showMsg('prompt_msg','<%=rb.getString("VLAN2MingChengGeShiBuZhengQue")%>');
						return;
					}
					//都通过，则输入正确
					var flag = apntype2 + enable2 + df2;
					paramVlan = paramVlan  + "(" + apnname2 + "|"+ flag + "|" + ip2 + "|" + vlan2.replace(/,/g," ") + ")";
					vlanList = vlanList + "," +vlan2;
					
				}else {
					if(!apnname2 && (vlan2.length > 0) ){
						alertFlag = true;
						showMsg('error_msg','<%=rb.getString("QingShuRuApnName2")%>');
						return;
					} 
					if(!vlan2 && (apnname2.length > 0) ){
						alertFlag = true;
						showMsg('error_msg','<%=rb.getString("QingShuRuVlanList2")%>');
						return;
					}
					paramVlan = paramVlan + "()";
				}
				
			}
		} else {
			paramVlan = paramVlan + "()";
		}
		
		var apnname3 = $("#apnname3").val();
		var vlan3 = $("#vlan3").val();
		var enable3 = $("#apnenable3").get(0).checked ? "1" : "0";
		var apntype3 = $("#apntype3").val();
		var df3 = $("#defaultrouter3").get(0).checked ? "1" : "0";
		var ip3 = $("#apnip3").val();
		if(!ip3){
			ip3="";
		}
		if(enable3 == "1"){
			var apnFlag = validateApnname(apnname3);
			var vlanFlag = validateVlan(vlan3);
			if(apntype3 == "000"){//L3 只需验证APNName + defaultrouter
				L3Num = L3Num + 1;//记录L3个数
				if(apnname3){
					if(!apnFlag){
						alertFlag = true;
						showMsg('prompt_msg','<%=rb.getString("APN3MingChengGeShiBuZhengQue")%>');
						return;
					}
					if(df3== "1"){//df选中
						L3dfNum = L3dfNum + 1;
					}
					var flag = apntype3 + enable3 + df3;
					paramVlan = paramVlan  + "(" + apnname3 + "|" + flag + "|" + ip3 + "|" + ")";
					
				} else {
					alertFlag = true;
					showMsg('prompt_msg','<%=rb.getString("QingShuRuApnName3")%>');
					return;
				}
			} else {//L2
				if(apnname3 && vlan3){
					//都存在输入，则校验
					var apnFlag = validateApnname(apnname3);
					var vlanFlag = validateVlan(vlan3);
					if(!apnFlag){
						alertFlag = true;
						showMsg('prompt_msg','<%=rb.getString("APN3MingChengGeShiBuZhengQue")%>');
						return;
					}
					if(!vlanFlag){
						alertFlag = true;
						showMsg('prompt_msg','<%=rb.getString("VLAN3MingChengGeShiBuZhengQue")%>');
						return;
					}
					//都通过，则输入正确
					var flag = apntype3 + enable3 + df3;
					paramVlan = paramVlan  + "(" + apnname3 + "|"+ flag + "|" + ip3 + "|" + vlan3.replace(/,/g," ") + ")";
					vlanList = vlanList + "," + vlan3;
					
				}else {
					if(!apnname3 && (vlan3.length > 0) ){
						alertFlag = true;
						showMsg('prompt_msg','<%=rb.getString("QingShuRuApnName3")%>');
						return;
					} 
					if(!vlan2 && (apnname3.length > 0) ){
						alertFlag = true;
						showMsg('prompt_msg','<%=rb.getString("QingShuRuVlanList3")%>');
						return;
					}
					paramVlan = paramVlan + "()";
				}
				
			}
		} else {
			paramVlan = paramVlan + "()";
		}
		
		
		var apnname4 = $("#apnname4").val();
		var vlan4 = $("#vlan4").val();
		var enable4 = $("#apnenable4").get(0).checked ? "1" : "0";
		var apntype4 = $("#apntype4").val();
		var df4 = $("#defaultrouter4").get(0).checked ? "1" : "0";
		var ip4 = $("#apnip4").val();
		if(!ip4){
			ip4="";
		}
		if(enable4 == "1"){
			var apnFlag = validateApnname(apnname4);
			var vlanFlag = validateVlan(vlan4);
			if(apntype4 == "000"){//L3 只需验证APNName + defaultrouter
				L3Num = L3Num + 1;//记录L3个数
				if(apnname4){
					if(!apnFlag){
						alertFlag = true;
						showMsg('prompt_msg','<%=rb.getString("APN4MingChengGeShiBuZhengQue")%>');
						return;
					}
					if(df2 == "1"){//df选中
						L3dfNum = L3dfNum + 1;
					}
					var flag = apntype4 + enable4 + df4;
					paramVlan = paramVlan  + "(" + apnname4 + "|" + flag + "|" + ip4 + "|" + ")";
					
				} else {
					alertFlag = true;
					showMsg('prompt_msg','<%=rb.getString("QingShuRuApnName4")%>');
					return;
				}
			} else {//L2
				if(apnname4 && vlan4){
					//都存在输入，则校验
					var apnFlag = validateApnname(apnname4);
					var vlanFlag = validateVlan(vlan4);
					if(!apnFlag){
						alertFlag = true;
						showMsg('prompt_msg','<%=rb.getString("APN4MingChengGeShiBuZhengQue")%>');
						return;
					}
					if(!vlanFlag){
						alertFlag = true;
						showMsg('prompt_msg','<%=rb.getString("VLAN4MingChengGeShiBuZhengQue")%>');
						return;
					}
					//都通过，则输入正确
					var flag = apntype4 + enable4 + df4;
					paramVlan = paramVlan  + "(" + apnname4 + "|"+ flag + "|" + ip4 + "|" + vlan4.replace(/,/g," ") + ")";
					vlanList = vlanList + "," + vlan4;
					
				}else {
					if(!apnname4 && (vlan4.length > 0) ){
						alertFlag = true;
						showMsg('prompt_msg','<%=rb.getString("QingShuRuApnName4")%>');
						return;
					} 
					if(!vlan4 && (apnname4.length > 0) ){
						alertFlag = true;
						showMsg('prompt_msg','<%=rb.getString("QingShuRuVlanList4")%>');
						return;
					}
					paramVlan = paramVlan + "()";
				}
				
			}
		} else {
			
			paramVlan = paramVlan + "()";
		}

		 
		//四个vlan最少配置一个APN,一个L3类型 ,
		var enableNum=0;
		var vlanL3Num=0;
		var defaultRouterNum = 0;
		$(".apnType_col select").each(function(){
			if($(this).val()=='000' && !$(this).is(":disabled")){
				vlanL3Num+=1;
				var vlan3ClassName = $(this).parent()[0].className.split(" ")[0];
				var defaultRouterEle = $("."+vlan3ClassName+".defaultRouter_row").children(":first-child");
				if(defaultRouterEle.is(":checked")){
					defaultRouterNum+=1;
				}
			}
		});
		$(".enable_col input").each(function(){
			if($(this).is(":checked")){
				enableNum+=1;
			}
		})
		if(enableNum<1){
			showMsg('prompt_msg','<%=rb.getString("ZhiShaoPeiZhiYiZuApn")%>');
			alertFlag = true;
			return;
		}
		if(vlanL3Num<1){
			showMsg('prompt_msg','<%=rb.getString("ZhiShaoXuanZeYiZuL3")%>');
			alertFlag = true;
			return;
		}
		if(defaultRouterNum>1 || (vlanL3Num>=1&&defaultRouterNum==0)){
			showMsg('prompt_msg','<%=rb.getString("DefaultRouterL3ZhiShaoXuanZeYiGe")%>');
			alertFlag = true;
			return;
		}
		if(paramVlan.length < 16){
			alertFlag = true;
			showMsg('prompt_msg',"<%=rb.getString("ZhiShaoShuRuYiZuShuJu")%>");
			return;
		}
		
		paramVlan = "[" + paramVlanHeader + paramVlan  + "]";
		if((apnname1==apnname2 && apnname1 !="") ||(apnname1==apnname3  && apnname1 !="") ||(apnname1==apnname4  && apnname1 !="") ||(apnname2==apnname3  && apnname2 !="") ||(apnname2==apnname4  && apnname2 !="") ||(apnname3==apnname4  && apnname3 !="") ){
			alertFlag = true;
			showMsg('prompt_msg',"<%=rb.getString("ApnNameChongFu")%>");
			return;
		}
		
		
		if(vlanList.split(",").length > 14){
			alertFlag = true;
			showMsg('prompt_msg',"<%=rb.getString("VlanZuiDuoXianZhi")%>");
			return;
		}
		
		var tempVlan = vlanList.split(",");
		var flagv=true; 
				
		for(var i =0;i<tempVlan.length && flagv;i++){
			for(var j =0;j<tempVlan.length;j++){
				if(tempVlan[i] == tempVlan[j] && (j!=i)){
					flagv = false;
					break;
				}
			}
		}
		if(!flagv){
			alertFlag = true;
			showMsg('prompt_msg',"<%=rb.getString("VlanChongFu")%>");
			return;
		}
		
		params["paramVlan"] = paramVlan;
		
		
		return paramVlan;	 
	}
	function vlanListChangeRed(ele){
		$(ele).keyup(function(){
			var reg = /^(\d+,?)+$/;
		    if (reg.test($(ele).val()) || !$(ele).val()) {
		        var arrVlan = $(ele).val().split(",");
		        if(arrVlan.length>16){
		        	/* $(ele).css('border-color','#E64242'); */	        	
			        $($(ele).next()).css('color','#E64242');
		        }else{	        	
			    	for(var i in arrVlan){
			    		if(typeof arrVlan[i] != 'function'){
				    		if(((arrVlan[i]-0) < 4 || (arrVlan[i]-0) > 4094)){
				    			/* $(ele).css('border-color','#E64242'); */
				    	        $($(ele).next()).css('color','#E64242');
				    		}else{
				    	    	/* $(ele).css('border-color','initial'); */
				    	        $($(ele).next()).css('color','#D2D2D2');
				    		}
			    		}	    		
			    	}
		        }
		    } else {
		        /* $(ele).css('border-color','#E64242'); */
		        $($(ele).next()).css('color','#E64242');
			} 
		})
	}
	function apNameChangeRed(ele){
		$(ele).keyup(function(){
			var reg = /^([a-zA-Z_])+([a-zA-Z0-9_])*$/;
		    if (reg.test($(ele).val()) || !$(ele).val()) {
		    	/* $(ele).css('border-color','initial'); */
		        $($(ele).next()).css('color','#D2D2D2');
		    } else {
		        /* $(ele).css('border-color','#E64242'); */
		        $($(ele).next()).css('color','#E64242');
			} 
		})
	}
	function ipAddressChangeRed(ele){
		$(ele).keyup(function(){
			var reg = /^(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])$/ ;
		    if (reg.test($(ele).val()) || !$(ele).val() ) {
		        $($(ele).next()).css('color','#D2D2D2');	    	
		    }else{
	    		$($(ele).next()).css('color','#E64242');	    	
		    }
		})    
	}
	/*设置完成提交*/	
	var alertFlag = false;
	function apnCommit(){
		alertFlag = false;
		var params = {
				cpeCode : cpeCode,
				timeZone : timeZone
		}
 		var qoschange = validateQoSIsChange();
 		if(!qoschange){
 			showMsg('prompt_msg','<%=rb.getString("CanShuZhiMeiYouBianHua")%>');
 			return;
 		}else{
 			var qosparam = setQos();
 			if(!qosparam){
 				return;
 			}
 			params["qos"] = qosparam;
 		}
		if(params["qos"]){
	 		if(!alertFlag){
	 			progressDivShow();
	 	 	}
	 		$("#cpeApnDiv").addClass("loading");
		 	$.post("${ctx}/cell/CPE/setAPN.action", params, function(data){
		 		if(data["success"]){
		 			progressDivHide();
		 			cpevm.$refs.setting.hide();
		 			cpevm.refreshList();
		 		}else{
		 			showMsg('error_msg',data["message"]);
		 		}
		 		$("#cpeApnDiv").removeClass("loading");
			}, "json");
	 	}
	}
	function closeApnSet(){
		$("#cpeSettingOption").animate({right:'-1700px'},500,function(){
			$("#cpeSettingOption").html("");
		});
	}

	//APN复选框点击事件
	function apnCheckFun(ele){
		var operateClassName = $(ele).parent()[0].className.split(" ")[0];
		if(!$(ele).is(":checked")){
			$("."+operateClassName).each(function(){
	            $($(this)[0]).children(":first-child").prop("disabled","disabled");
	        })
	        $(ele).prop("disabled","");
	        $(ele).prop("cheched","cheched");
		}else{
			$("."+operateClassName).each(function(){
	            $($(this)[0]).children(":first-child").prop("disabled","");
	        })
	        var apnTypeFlag=$("."+operateClassName+".apnType_row").children(":first-child").val();
			if(apnTypeFlag=='001'){
				$("."+operateClassName+".defaultRouter_row").children(":first-child").prop("disabled","disabled");	
			}else{
				$("."+operateClassName+".vlanList_row").children(":first-child").prop("disabled","disabled");			
			}
		}
		$(".apnIp_col input").prop("disabled","disabled");
	}

	//APN下拉框改变事件
	function apnSelectFun(ele){
		var operateClassName = $(ele).parent()[0].className.split(" ")[0];
		if($(ele).val()=='001'){
			$("."+operateClassName+".vlanList_row").children(":first-child").prop("disabled","");
			$("."+operateClassName+".defaultRouter_row").children(":first-child").prop("checked","");
			$("."+operateClassName+".defaultRouter_row").children(":first-child").prop("disabled","disabled");		
			$("."+operateClassName+".apnName_row").children(":first-child").val("");
			$("."+operateClassName+".apnName_row").children(":last-child").css('color','#D2D2D2');
			$("."+operateClassName+".vlanList_row").children(":first-child").val("");
			$("."+operateClassName+".vlanList_row").children(":last-child").css('color','#D2D2D2');
		}else{
			$("."+operateClassName+".apnName_row").children(":first-child").val("");
			$("."+operateClassName+".apnName_row").children(":last-child").css('color','#D2D2D2');
			$("."+operateClassName+".vlanList_row").children(":first-child").val("");
			$("."+operateClassName+".vlanList_row").children(":last-child").css('color','#D2D2D2');
			$("."+operateClassName+".vlanList_row").children(":first-child").prop("disabled","disabled");
			$("."+operateClassName+".defaultRouter_row").children(":first-child").prop("disabled","");
		}
	}
</script>