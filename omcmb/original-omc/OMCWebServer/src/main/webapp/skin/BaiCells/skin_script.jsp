<%@ page language="java" contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp" %>
<style>
	.sso-ctner {
		width: 80%;
		margin: 0 auto;
	}
	.sso-bt-wrapper {
		width: 100%;
		margin: 3vh auto;
	}
	.sso-bt {
		display: inline-block;
		padding: 8px 20px;
		border: 1px solid #6a64a5;
		border-radius: 5px;
		cursor: pointer;
		color: #6a64a5;
		background-color: #f2f2f2;
	}
	.sso-bt:hover {
		background-color: #fff;
	}
	.split-title {
		width: 100%;
		display: flex;
		align-items: center;
		justify-content: center;
	}
	.split-title::before,
	.split-title::after {
		content: '';
		display: inline-block;
		border-bottom: 1px dashed #b0b0b0;
		width: 20%;
		flex: auto;
	}
</style>
<div class="lockPage" id="winLockScreen" style="display:none;">
<div class="lockbody">
	<div class='lockIconItem'>
   		<div style="margin:auto;display:flex">
   			<p style='line-height:40px;'><span class="el-icon el-icon-circle-warning" style="margin-right:10px;font-size:16px;"></span><span style="font-size:14px;"><%=rb.getString("JianRongTiShi")%>:</span></p>
   			<div style="display:inline-block;margin-left:10px">
   				<span class="browser_icon google_icon"></span>
   				<span class="browser_icon safari_icon"></span>
   				<span class="browser_icon firefox_icon"></span>
   				<span class="browser_icon edge_icon"></span>
   			</div>
   		</div>
    </div>
    <div class="lock_header"></div>
    <div class="lock_body">
		<div class="lockformbox">
			<form id="lockform">  
				<input type="hidden" name="userInfo.logintype" value="0" />
				<div class="lock_bar_logo_div"></div>
				<div class="lock_user_div">
					<div class="input_ico_user_div_lock el-icon el-icon-menu-subscriber"></div>
					<input style="color:#96b3ed;width:80%;" readonly type="text" id="uid_lock" class="input_lock" name="userInfo.usercode"  value="${userInfo.usercode}">
				</div>
				<div v-if="!isSupportSSO" class="lock_pwd_div">
					<div class="input_ico_pwd_div_lock el-icon el-icon-common-lock"></div>
					<input type="password" style="display:none">
					<div v-if="isBrowserAutoRecordPass" style="display: inline-block;width: 80%;">
						<el-password style="width: 100%;" v-model="password" @enter="passwordEnterClick" size="mini" @input="passwordChange" placeholder='<%=rb.getString("QingShuRuMiMa")%>'></el-password>
						<input id="lockPassword" style="width:100%;display:none;" type="text" class="input_lock" name="userInfo.password" placeholder="<%=rb.getString("QingShuRuMiMa")%>">
					</div>
					<input v-if="!isBrowserAutoRecordPass" style="width: 80%;" id="lockPassword" type="password" class="input_lock" name="userInfo.password" placeholder="<%=rb.getString("QingShuRuMiMa")%>">
					<span id="lockPassResult" style="color:red; font-size: 13px; display:block;text-align:left;margin-left:41px;margin-top:8px;"></span>
				</div>
				<div v-if="!isSupportSSO" class="lock_btn_div" onclick="javascript:cancelLock_RSA()">
					<span style="line-height: 38px;	font-size: 20px; color: #FFFFFF;"><%=rb.getString("JieSuo")%></span>
				</div>
			</form>

			<div v-if="isSupportSSO" class="sso-ctner">
				<div class="split-title">
					<span style="display: inline-block; padding: 0px 10px;color: #636363;">Unlock using SSO</span>
				</div>
				<div class="sso-bt-wrapper">
					<div class="sso-bt" @click="submit_sso">SSO</div>
				</div>
				<div style="display: none;">
					<form id="ssologinfrom" action="${ctx }/sso/login/sso.action" method="post">
						<input type="hidden" id="sso_user_code" name="userInfo.usercode" value="${userInfo.usercode}"/>
						<input type="hidden" id="sso_user_pwd" name="userInfo.password" value="${userInfo.password}"/>

						<input id="sso_btn_submit" type="submit" style="display: none;"/>
					</form>
				</div>
			</div>
		</div>
	</div>

</div> 
</div>
 	
<!-- 新文件版本升级通知 -->
 <div class='newFileInfoAlert' style='display:none'>
	<div style='height:35px;' class='newFileTitle' onclick="showMoreMes(event)">
		<span class='upgradeNumber'></span>
		<span class='upgrageTitle'><%=rb.getString("YouKeShengJiWenJianDianJiChaKanXiangQing")%></span>
		<span id='showMoreMesImg' class='upgrade_down arrowUpgrade flag'></span>
		<span class='ignoreAll' onclick='NoNoticeAll(event)'><%=rb.getString("Button_HuLueSuoYouBanBen") %></span>
		<span class='el-icon el-icon-close' onclick='closeMoreMes(event)' style='display:inline-block;width:20px;height:20px;margin-left:20px;margin-top:8px;'/>
	</div>
	<div id='newFileDetails' class='newUpGradeFileDetails' style='max-height:500px;overflow:auto;display:none;padding-bottom:20px;'></div>
</div>
<div id='upgrade_container' style='display:none;position:absolute;left:0px;right:0px;bottom:0px;top:47px;background:#fff;z-index:1000;overflow-y:auto;width:100%;'>	
</div>

<div id="mainCover" class="window-mask" style="display:none;width:2000px;height:1800px;position:absolute;left:0px;top:0px;z-index:2000"></div>
<div id="modal" class="modal"></div>
    
<div id="getCellneedRebootTips" class="getCellInfo" style="display: none;">
	<span></span>
</div>

<div id="epcWinLoadingPro"  style="position:absolute;display:none;width:100%;height:100%;z-index:20000;background:#CDE0E8;opacity:0.5;"> 
	<div id="loadingCover" style="display:none;position:absolute;z-index:200000;margin-left:48%;margin-top:20%;">	    
		    <img src="${ctx}/css/images/global/loading.gif" />
	</div>
</div>
<script>
	var baiCellsLockVue = new Vue({
			el:'#winLockScreen',
			data(){
				return{
					password:'',
					isBrowserAutoRecordPass:false
				}
			},
			computed: {
				isSupportSSO() {
					return isSupportSSO == true || isSupportSSO == 'true';
				}
			},
			methods:{
				init(){
					var vm = this;
					$.ajax({
						type: "post",
						url: '${ctx}/sys/login/getIsBrowserAutoRecordPass',
						data: {},
						async: false,
						dataType:"json",
						success: function(data) {
							vm.isBrowserAutoRecordPass = data ? true : false;
						}
					});
				},
				// 密码改变事件
				passwordChange(val){
					var vm = this;
					$("#lockPassword").val(val)
				},
				// 密码回车事件
				passwordEnterClick(){
					cancelLock_RSA();
					return false;
				},
				// SSO 登录
				submit_sso() {
					$("#sso_btn_submit").click();
				}
			},
			mounted(){
				this.init();
			}
		})
</script>
 <script type="text/javascript">
 
	//postmessage 解决跨域
	function reviceMessageFromBss (event){
		if(event.data.msg == "messomc"){
			documentClick();
		}
	}
	window.addEventListener("message",reviceMessageFromBss);

	function cancelLock_RSA() {
		if($("#lockPassword").val() == ""){
			$('#lockPassResult').text(QingShuRuMiMa);
		}
		if ($("#lockPassword").val() != "") {
			var dateStr = dateformatter(new Date()).replace(' ','-').substr(3);
			
			$.post(webRootPath + "/system/sysuser/doCancelLockScreen.action", {
				lockPassword: RsaEncrypt($("#lockPassword").val(), '${publicKey}'), 
				rd: AesEncrypt(dateStr)}, 
				function (data) {
				var result = AesDecrypt(data.result||'4yNThc1APBpYgFK4s6OvOw==', CryptoJS.enc.Utf8.parse(dateStr));
				
				if(result.success == true) {
					lastVisitedTime = new Date(gloableTime);
					$("#lockPassword").val("");
					$('#lockPassResult').text("");
					$("#winLockScreen").css("display","none");
					sessionStorage.locked = '';
				}else {
					$('#lockPassResult').text(result["message"]);
					$("#lockPassword").val("");
				}
				return;
			}, "json");
		}
	}
	
    $(function () {
		$.post("${ctx}/ui/customization/getCustomizationInfo",{},function(data){
			if(data["ui_restore"] == "true"){
				
			}else {
				//如果用户自定义的了登录背景图，那么隐藏标语样式，去除baicells字样
				if(data.ui_login_background){
					$(".lock_header").hide()
				}else{
					$(".lock_header").show();
				}
			}
		},"json")
    	
    	//扩展easyui控件
    	extendEasyui();

    	// 点击页面其他地方时，如果右上角的用户弹窗是显示的，则将其隐藏
    	$(document).click(function(e) {
    		documentClick();
    		if($(e.target).closest(".newFileInfoAlert").length==0){
    			$("#newFileDetails").slideUp(400);
        		$("#newFileDetails .upgrade_detail_mes").map(function(index,item){
        			$(item).slideUp(400,function(){
        				$(item).prev().find(".upgrade_view").html("<%=rb.getString("ChaKan")%>");
        			});
        		})
            	$("#showMoreMesImg").removeClass("upgrade_up").addClass("upgrade_down");
            	slideFlag = true;
    		}
    	});
        combonFirstTimeAndLeftMenu();
		
        /*是否有密码将过期提醒 */
        //判断是否是4A系统
        if (isCloudCore == "false") {
	        $.post("${ctx}/system/sysuser/queryPaswdExpirePrompt.action", {}, function (data) {
	            if (data["Expire"].length > 0 && "${ispwdtitle}"!="true" && data["pwd_enable"] == "0") {
            		/* $.messager.alert(TiShi, data["Expire"]); */
            		showMsg('prompt_msg',data["Expire"]);
	                return;
	            }
	        }, "json");
        }
        
        //判断是否是兼容浏览器
        var explorer = window.navigator.userAgent;
		var explorFlag = false;
		if(explorer.indexOf("Firefox") > 0){//火狐浏览器
			explorFlag = true;
		}else if(explorer.indexOf("Edg") > 0){//Edge浏览器
			explorFlag = true;
		}else if(explorer.indexOf("Mac") > 0){//Safari浏览器
			explorFlag = true;
		}else if(isChrome()){//Google浏览器
			explorFlag = true;
			// 完全使用chromiun内核，且不做任务差异处理的
			try{
				var mimeTypes = Array.from(window.navigator.mimeTypes).map(function(item){ return item.type;}),
					excludes = ['application/360softmgrplugin','application/x-ppapi-widevine-cdm'];
				
				excludes.map(function(item){
					if(mimeTypes.includes(item)) explorFlag = false;
				})
			}catch(e){}
		}else{
			explorFlag = false;
		}
		if(explorFlag){//此浏览器是支持的浏览器
			$(".lockIconItem").hide();
		}else{
			$(".lockIconItem").show();
		}
    });  
    function isChrome(){
		var agent = window.navigator.userAgent,
			reg = /^[\s\S]+Chrome\/[0-9\. ]+(\sMobile\s)?Safari\/[0-9\.]+$/;
	
		return reg.test(agent);
	}
    
    var fileLength = '';
    var fileDetail = [];
    function goVersionNotice(data){
    	// enodeb（基站全量升级）、enodeb_patch（基站补丁升级）、enodeb_uboot（基站UBOOT升级）、cpe_odu（CPE ODU升级）、cpe_idu（CPE IDU升级）
    	 if(!data||getObjLength(data)<=0){
    		 $(".newFileInfoAlert").hide();
    	}else{
    		var enodeb_upgrade = data.enodeb_upgrade;
        	var enodeb_patch_upgrade = data.enodeb_patch_upgrade;
        	var enodeb_fpga_upgrade = data.enodeb_fpga_upgrade;
        	var enodeb_uboot_upgrade = data.enodeb_uboot_upgrade;
        	var cpe_odu_upgrade = data.cpe_odu_upgrade;
        	var cpe_idu_upgrade = data.cpe_idu_upgrade;
        	
        	if(enodeb_upgrade){
        		enodeb_upgrade.map(function(item,index){
        			var obj = {};
        			obj.upgradeType = "enodeb";
        			obj.typeNumber = 0;
        			obj.typeName = "eNB";
        			obj.title = '<%=rb.getString("Msg_YouJiZhanRuanJianBanBenKeGengXin")%>&nbsp;';
        			obj.version = item.version;
        			obj.versionId = item.versionId;
        			obj.desc = item.description;
        			obj.cls = "CODE_ENB_UPGRADE_IMAGE hidden";
        			obj.file_type = item.file_type;
        			obj.product = item.product;
        			fileDetail.push(obj);
        		})
        	}
        	if(enodeb_patch_upgrade){
        		enodeb_patch_upgrade.map(function(item,index){
        			var obj = {};
        			obj.upgradeType = "enodeb_patch";
        			obj.typeNumber = 1;
        			obj.typeName = "eNB";
        			obj.title = "<%=rb.getString("Msg_YouJiZhanPATCHRuanJianBanBenKeGengXin")%>&nbsp;";
        			obj.version = item.version;
        			obj.versionId = item.versionId;
        			obj.desc = item.description;
        			obj.cls = "CODE_ENB_UPGRADE_PATCH hidden";
        			obj.file_type = item.file_type;
        			obj.product = item.product;
        			fileDetail.push(obj);
        		})
        	}
         	if(enodeb_uboot_upgrade){
         		enodeb_uboot_upgrade.map(function(item,index){
        			var obj = {};
        			obj.upgradeType = "enodeb_uboot";
        			obj.typeNumber = 2;
        			obj.typeName = "eNB";
        			obj.title = "<%=rb.getString("Msg_YouJiZhanUBOOTRuanJianBanBenKeGengXin")%>&nbsp;";
        			obj.version = item.version;
        			obj.versionId = item.versionId;
        			obj.desc = item.description;
        			obj.cls = "";
        			obj.file_type = item.file_type;
        			obj.product = item.product;
        			fileDetail.push(obj);
        		})
        	}
         	if(enodeb_fpga_upgrade){
         		enodeb_fpga_upgrade.map(function(item,index){
        			var obj = {};
        			obj.upgradeType = "enodeb_fpga";
        			obj.typeNumber = 6;
        			obj.typeName = "eNB";
        			obj.title = "<%=rb.getString("Msg_YouJiZhanFPGARuanJianBanBenKeGengXin")%>&nbsp;";
        			obj.version = item.version;
        			obj.versionId = item.versionId;
        			obj.desc = item.description;
        			obj.cls = "CODE_ENB_UPGRADE_FPGA hidden";
        			obj.file_type = item.file_type;
        			obj.product = item.product;
        			fileDetail.push(obj);
        		})
        	}
         	if(cpe_odu_upgrade){
         		cpe_odu_upgrade.map(function(item,index){
        			var obj = {};
        			obj.upgradeType = "cpe_odu";
        			obj.typeNumber = 3;
        			obj.typeName = "CPE";
        			obj.title = "<%=rb.getString("Msg_YouCPERuanJianBanBenKeGengXin")%>&nbsp;";
        			obj.version = item.version;
        			obj.versionId = item.versionId;
        			obj.desc = item.description;
        			obj.cls = "CODE_CPE_UPGRADE_IMAGE hidden";
        			obj.file_type = "1";
        			fileDetail.push(obj);
        		})
        	}
         	if(cpe_idu_upgrade){
         		cpe_idu_upgrade.map(function(item,index){
        			var obj = {};
        			obj.upgradeType = "cpe_idu";
        			obj.typeNumber = 4;
        			obj.typeName = "CPE";
        			obj.title = "<%=rb.getString("Msg_YouCPERuanJianBanBenKeGengXin")%>&nbsp;";
        			obj.version = item.version;
        			obj.versionId = item.versionId;
        			obj.desc = item.description;
        			obj.cls = "CODE_CPE_UPGRADE_IMAGE hidden";
        			obj.file_type = "2";
        			fileDetail.push(obj);
        		})
        	}
         	fileLength = fileDetail.length;
         	$(".upgradeNumber").html(fileLength);
         	fileDetail.map(function(item,index){
         		var newDesc = item.desc.replace(/[\r\n]/g,"");
         		newDesc = newDesc.replace(/\'/g,'#quot').replace(/\"/g,'@quot');
         		var data  = JSON.stringify(item);
         	  	var infoDetail = "<div versionId="+item.versionId+"  class='upgrade_file_title' style='margin-left:30px;margin-top:20px;'>";
            	infoDetail+="<div class='upgrade_oper' style='display:flex;flex-direction:row'>";
            	infoDetail+="<span class='typeUpgrade'>"+item.typeName+"</span>";
                infoDetail+="<span class='upgradeTip'>"+item.title+item.version+"</span>";
                infoDetail+="<div><span onclick='viewMoreDetail(this,\""+newDesc+"\",\""+item.versionId+"\",\""+item.upgradeType+"\",\""+item.file_type+"\")' class='upgrade_view' style='border-bottom:1px solid #4C6778;'>"+"<%=rb.getString("ChaKan")%>"+"</span>";
                infoDetail+="<span onclick='NoNotice(this,\""+item.upgradeType+"\",\""+item.version+"\",\""+item.versionId+"\")' class='noNotice' style='border-bottom:1px solid #4C6778;margin-left:10px;'>"+"<%=rb.getString("Button_HuLue")%>"+"</span>";
                infoDetail+="<span onclick='goUpgrade("+data+")' class='upgrade "+item.cls+"' style='border-bottom:1px solid #4C6778;margin-left:10px;'>"+"<%=rb.getString("ShengJi")%>"+"</span>";
                
                infoDetail+="</div></div>";
                infoDetail+="<div class='upgrade_detail_mes' style='display:inline-block'></div></div>";
                $("#newFileDetails").append(infoDetail);
            	if(language == 'en'){
            		$(".upgradeTip").css("width","470px");
            	}else{
            		$(".upgradeTip").css("width","360px");
            	}
         	})
         	 $(".newFileInfoAlert").show();
        }
    }
    function NoNoticeAll(e){
    	e.stopPropagation();
    	 $.post("${ctx}/system/sysuser/goCloseUpgradePrompt.action", {}, function(data){
             if(data.success === true){                  
                 $(".newFileInfoAlert").hide();
             }
         }, "json");
    }
    function NoNotice(ele,type,version,versionId){
    	var param = {
    		upgrade_type:type,
    		version:version,
			versionId:versionId
    	}
    	$.post("${ctx}/system/sysuser/goCloseUpgradePrompt.action", param, function(data){
            if(data.success === true){
                $($(ele).parent().parent()).remove(); 
            	if(fileDetail.length > 0){
            		fileDetail.map(function(item,index){
                		if(item.versionId == versionId){
                			fileDetail.splice(index,1);
                		}
                	})
            	}
                $(".upgradeNumber").html(fileDetail.length);
            	if($("#newFileDetails").find(".upgrade_file_title").length == 0){
            		$("#newFileDetails").slideUp(400);  
                	$("#showMoreMesImg").removeClass("upgrade_up").addClass("upgrade_down");
                	slideFlag = true;
                	$(".newFileInfoAlert").hide();
            	}
            	//先点击 view,再点击 Ignore 时，将详情内容也隐藏
            	$("#newFileDetails .upgrade_detail_mes").hide();
            }
        }, "json");
    }
    var slideFlag = true;
    function showMoreMes(e){
    	e.stopPropagation();
    	if(slideFlag){
    		$("#newFileDetails").slideDown(400);
        	$("#showMoreMesImg").removeClass("upgrade_down").addClass("upgrade_up");
        	slideFlag = false;
    	}else{
    		$("#newFileDetails").slideUp(400);
    		$("#newFileDetails .upgrade_detail_mes").map(function(index,item){
    			$(item).slideUp(400,function(){
    				$(item).prev().find(".upgrade_view").html("<%=rb.getString("ChaKan")%>");
    			});
    		})
        	$("#showMoreMesImg").removeClass("upgrade_up").addClass("upgrade_down");
        	slideFlag = true;
    	}
    }
    function closeMoreMes(e){
    	e.stopPropagation();
    	$("#newFileDetails").slideUp(400);  
    	$("#showMoreMesImg").removeClass("upgrade_up").addClass("upgrade_down");
    	slideFlag = true;
    	$(".newFileInfoAlert").css("display","none");
    }
    function viewMoreDetail(ele,desc,versionId,upgradeType,file_type){
    	desc = desc.replace(/#quot/g,'\'').replace(/@quot/g,'\"');
    	
    	if( $(ele).closest(".upgrade_file_title").siblings().length == 0){
    		var others = '';
    	}else{
    		others = $(ele).closest(".upgrade_file_title").siblings().find(".upgrade_detail_mes");
    	}
    	 if($(ele).html()=="<%=rb.getString("ChaKan")%>"){
    		$(ele).parents(".upgrade_oper").next().html("");
    		var infoDetail = "<div style='margin-top:10px;'>";
    		infoDetail+="<p>--"+RuanJianBanBenXiangXiXinXi+"</p>";
    	    infoDetail+="<p class='upgrade_desc'>"+desc+"</p>";
    	    var onclick='toSoftwareFile("'+versionId+'","'+upgradeType+'","'+file_type+'")';
    	    infoDetail+="<p style='cursor:pointer;color:#1DA3FC;margin-top:5px;margin-left:10px;' onclick='"+onclick+"'><%=rb.getString("XiangXiXinXin")%>>></p>";
    	    infoDetail+="</div>";
    	    $(ele).parents(".upgrade_oper").next().append(infoDetail);
            $(ele).html("<%=rb.getString("GuanBi")%>");
           	var slideDiv =  $(ele).parents(".upgrade_oper").next();
            if(others){
            	others.each(function(index,item){
                   	$(item).slideUp(300);
                   	slideDiv.slideDown(400);
                   	$(item).prev().find(".upgrade_view").html("<%=rb.getString("ChaKan")%>");
                })
            }else{
            	$(ele).parents(".upgrade_oper").next().slideDown(400);
            }
    	}else{
    		$(ele).parents(".upgrade_oper").next().slideUp(400);
    		 $(ele).html("<%=rb.getString("ChaKan")%>");
    	}
    }
  	function toSoftwareFile(versionId,upgradeType,file_type){
  		 if(upgradeType == "enodeb"){
    		 isJumpToPage = {code:'enodeb',vid:versionId,type:'view',file_type:file_type}
   			 try{
	   			sessionStorage.setItem('submenuid', '500001');
	   			sysMain.headType = 'enb';
   				eventAllBus.$emit("gomenupage","5000","","5000",false, function(){});
   				//#69352
   				if(enbFileVue) {
   				    //点击view按钮tab选中，关闭新建任务页面
                    enbFileVue.activeName = 'file';
                    if(isJumpToPage.file_type == "0") enbFileVue.file_type = "upgrade";
                    if(isJumpToPage.file_type == "1") enbFileVue.file_type = "ca";
                    if(isJumpToPage.file_type == "6") enbFileVue.file_type = "fpga";
                    enbFileVue.changeFileType(enbFileVue.file_type);
                    enbFileVue.$refs.slide.hide();
                }
   			 }catch(e){}
    	  } 
  		 if(upgradeType == "enodeb_patch"){
  			isJumpToPage = {code:'enodeb_patch',vid:versionId,type:'view',file_type:file_type}
			 try{
	   			sessionStorage.setItem('submenuid', '500001');
	   			sysMain.headType = 'enb';
				eventAllBus.$emit("gomenupage","5000","","5000",false, function(){});
				if(enbFileVue) {
                    //点击view按钮tab选中，关闭新建任务页面
                    enbFileVue.activeName = 'file';
                    if(isJumpToPage.file_type == "0") enbFileVue.file_type = "upgrade";
                    if(isJumpToPage.file_type == "1") enbFileVue.file_type = "ca";
                    if(isJumpToPage.file_type == "6") enbFileVue.file_type = "fpga";
                    enbFileVue.changeFileType(enbFileVue.file_type);
                    enbFileVue.$refs.slide.hide();
                }
 			 }catch(e){}
  		 }
  		 if(upgradeType == "enodeb_uboot"){
  			isJumpToPage = {code:'enodeb_uboot',vid:versionId,type:'view',file_type:file_type}
			try{
   				sessionStorage.setItem('submenuid', '500001');
   				sysMain.headType = 'enb';
				eventAllBus.$emit("gomenupage","5000","","5000",false, function(){});
				if(enbFileVue) {
                    //点击view按钮tab选中，关闭新建任务页面
                    enbFileVue.activeName = 'file';
                    if(isJumpToPage.file_type == "0") enbFileVue.file_type = "upgrade";
                    if(isJumpToPage.file_type == "1") enbFileVue.file_type = "ca";
                    if(isJumpToPage.file_type == "6") enbFileVue.file_type = "fpga";
                    enbFileVue.changeFileType(enbFileVue.file_type);
                    enbFileVue.$refs.slide.hide();
                }
			}catch(e){}
  		 }
  		if(upgradeType == "enodeb_fpga"){
  			isJumpToPage = {code:'enodeb_fpga',vid:versionId,type:'view',file_type:file_type}
			try{
   				sessionStorage.setItem('submenuid', '500001');
   				sysMain.headType = 'enb';

				eventAllBus.$emit("gomenupage","5000","","5000",false, function(){});
                if(enbFileVue) {
   				    //点击view按钮tab选中，关闭新建任务页面
                    enbFileVue.activeName = 'file';
                    if(isJumpToPage.file_type == "0") enbFileVue.file_type = "upgrade";
                    if(isJumpToPage.file_type == "1") enbFileVue.file_type = "ca";
                    if(isJumpToPage.file_type == "6") enbFileVue.file_type = "fpga";
                    enbFileVue.changeFileType(enbFileVue.file_type);
                    enbFileVue.$refs.slide.hide();
                }
			}catch(e){}
  		 }
  		 if(upgradeType == "cpe_odu"){
  			isJumpToPage = {code:'cpe_odu',vid:versionId,type:'view'}
			try{
	   			sessionStorage.setItem('submenuid', '500003');
	   			sysMain.headType = 'cpe';
				eventAllBus.$emit("gomenupage","5000","","5000",false, function(){});
				if(cpeUpgrade) {
                    //点击view按钮tab选中，关闭新建任务页面
                    cpeUpgrade.activeName = 'file';
                    cpeUpgrade.$refs.upgradeSlide.hide();
                }
			}catch(e){}
  		 }
  		 if(upgradeType == "cpe_idu"){
  			isJumpToPage = {code:'cpe_idu',vid:versionId,type:'view'}
			try{
	   			sessionStorage.setItem('submenuid', '500003');
	   			sysMain.headType = 'cpe';
				eventAllBus.$emit("gomenupage","5000","","5000",false, function(){});
				if(cpeUpgrade) {
                    //点击view按钮tab选中，关闭新建任务页面
                    cpeUpgrade.activeName = 'file';
                    cpeUpgrade.$refs.upgradeSlide.hide();
                }
			}catch(e){}
  		 }
  		$("#newFileDetails").slideUp(400);
		$("#newFileDetails .upgrade_detail_mes").map(function(index,item){
			$(item).slideUp(400,function(){
				$(item).prev().find(".upgrade_view").html("<%=rb.getString("ChaKan")%>");
			});
		})
    	$("#showMoreMesImg").removeClass("upgrade_up").addClass("upgrade_down");
    	slideFlag = true;
  	}
    function goUpgrade(data){
    	$("#newFileDetails").slideUp(300,function(){
    		 var upgradeType = data.upgradeType,
    		 file_type = data.file_type,
    		 product = data.product,
    		 versionId = data.versionId;
	    	 if(upgradeType.includes("enodeb")){
	    		 isJumpToPage = {vid:versionId,file_type:file_type,product:product,type:'upgrade'}
	   			 try{
	   				sessionStorage.setItem('submenuid', '500001');
	   				sysMain.headType = 'enb';
	   			 	eventAllBus.$emit("gomenupage","5000","","5000",false, function(){});

	   			 	//#69352
                    if(enbFileVue) {
                        //点击升级按钮方可进入
                        enbFileVue.activeName = 'upgrade';
                        enbFileVue.product = product;
                        enbFileVue.addUpgradeTask();
                    }
	   			 }catch(e){}
	    	 }else{
	    		 isJumpToPage = {vid:versionId,file_type:file_type,type:'upgrade'}
	   			 try{
		   			sessionStorage.setItem('submenuid', '500003');
		   			sysMain.headType = 'cpe';
	   			 	eventAllBus.$emit("gomenupage","5000","","5000",false, function(){});
	   			 	if(cpeUpgrade) {
                        //点击view按钮tab选中，关闭新建任务页面
                        cpeUpgrade.activeName = 'upgrade';
                        cpeUpgrade.addUpgradeTask();
                    }
	   			 }catch(e){}
	    	 }
    	});
    	$("#showMoreMesImg").removeClass("upgrade_up").addClass("upgrade_down");
    	slideFlag = true;
    }
    
    
	function ChangeLanguageByCloudcore (event){
		if(event.data.msg == "ChangeLanguageZh"){
			ChangeLanguage('zh');
		}else if(event.data.msg == "ChangeLanguageEn"){
			ChangeLanguage('en');
		} else if(event.data.msg == "omcExit"){
			//postmessage 跨域退出
			logout('${ctx}');
		}
	}
	window.addEventListener("message",ChangeLanguageByCloudcore);


$(function(){
	$("#headerAlarmQuery").click(function(){
		if($(event.target).attr("class") != "showheaderMMLOp" && ($(event.target).attr("class") != "operation_more")){
			$(".showheaderMMLOp").hide();
		}
	})
})
</script>