<%@ page language="java" contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp" %>
<div class="lockbody" id="winLockScreen" style="display:none;">
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
	<div class="lockmain">
		<div class="locklogo">HeMS</div>
		<div class="lockformbox">
			<form id="lockform"  style="width: 100%; height: 100%; text-align: center; padding-right: 5px;">  
				<input type="hidden" name="userInfo.logintype" value="0" />
				<div class="lock_bar_logo_div"></div>
				<div class="lock_user_div">
					<div class="input_ico_user_div_lock"></div>
					<input style="color:#96b3ed" readonly type="text" id="uid_lock" class="input_" name="userInfo.usercode"  value="${userInfo.usercode}">
				</div>
				<div class="lock_pwd_div">
					<div class="input_ico_pwd_div_lock"></div>
					<input type="password" style="display:none">
					<input id="lockPassword" type="password" class="input_" name="userInfo.password" placeholder="<%=rb.getString("QingShuRuMiMa")%>">
					<span id="lockPassResult" style="color:red; font-size: 13px; display:block;text-align:left;margin-left:41px;margin-top:8px;"></span>
				</div>
				<div class="lock_btn_div" onclick="javascript:cancelLock()">
					<span style="line-height: 38px;	font-size: 20px; color: #FFFFFF;"><%=rb.getString("JieSuo")%></span>
				</div>
			</form>
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
		<span class='upgrade_close' onclick='closeMoreMes(event)' style='display:inline-block;width:20px;height:20px;margin-left:20px;margin-top:8px;'/>
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


 <script type="text/javascript">
 
	//postmessage 解决跨域
	function reviceMessageFromBss (event){
		if(event.data.msg == "messomc"){
			documentClick();
		}
	}
	window.addEventListener("message",reviceMessageFromBss);
	
	
 	
    
    
    
    $(function () {
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
			reg = /^[\s\S]+Chrome\/[0-9\. ]+Safari\/[0-9\.]+$/;
	
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
        			fileDetail.push(obj);
        		})
        	}
         	fileLength = fileDetail.length;
         	$(".upgradeNumber").html(fileLength);
         	fileDetail.map(function(item,index){
         	  	var infoDetail = "<div versionId="+item.versionId+"  class='upgrade_file_title' style='margin-left:30px;margin-top:20px;'>";
            	infoDetail+="<div>";
            	infoDetail+="<span class='typeUpgrade'>"+item.typeName+"</span>";
                infoDetail+="<span class='upgradeTip'>"+item.title+item.version+"</span>";
                infoDetail+="<span onclick='goUpgrade(\""+item.typeNumber+"\",\""+item.versionId+"\")' class='upgrade "+item.cls+"' style='float:right;margin-right:7px;border-bottom:1px solid #4C6778;margin-top:-16px;'>"+"<%=rb.getString("ShengJi")%>"+"</span>";
                infoDetail+="<span onclick='NoNotice(this,\""+item.upgradeType+"\",\""+item.version+"\",\""+item.versionId+"\")' class='noNotice' style='margin-top:-16px;float:right;margin-right:25px;border-bottom:1px solid #4C6778;'>"+"<%=rb.getString("Button_HuLue")%>"+"</span>";
                infoDetail+="<span onclick='viewMoreDetail(this,\""+item.desc+"\",\""+item.versionId+"\",\""+item.upgradeType+"\")' class='upgrade_view' style='margin:-16px 25px 0px 25px;float:right;border-bottom:1px solid #4C6778;'>"+"<%=rb.getString("ChaKan")%>"+"</span>";
                infoDetail+="</div>";
                infoDetail+="<div class='upgrade_detail_mes'></div></div>";
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
    function viewMoreDetail(ele,desc,versionId,upgradeType){
    	if( $(ele).closest(".upgrade_file_title").siblings().length == 0){
    		var others = '';
    	}else{
    		others = $(ele).closest(".upgrade_file_title").siblings().find(".upgrade_detail_mes");
    	}
    	 if($(ele).html()=="<%=rb.getString("ChaKan")%>"){
    		$($($(ele).parent()[0]).next()[0]).html("");
    		var infoDetail = "<div style='margin-left:30px;margin-top:10px;'>";
    		infoDetail+="<p>--"+RuanJianBanBenXiangXiXinXi+"</p>";
    	    infoDetail+="<p class='upgrade_desc'>"+desc+"</p>";
    	    var onclick='toSoftwareFile("'+versionId+'","'+upgradeType+'")';
    	    infoDetail+="<p style='cursor:pointer;color:#1DA3FC;margin-top:5px;margin-left:10px;' onclick='"+onclick+"'><%=rb.getString("XiangXiXinXin")%>>></p>";
    	    infoDetail+="</div>";
    	    $($($(ele).parent()[0]).next()[0]).append(infoDetail);
            $(ele).html("<%=rb.getString("GuanBi")%>");
            var slideDiv = $($($(ele).parent()[0]).next()[0]);
            if(others){
            	others.each(function(index,item){
                   	$(item).slideUp(300);
                   	slideDiv.slideDown(400);
                   	$(item).prev().find(".upgrade_view").html("<%=rb.getString("ChaKan")%>");
                })
            }else{
            	$($($(ele).parent()[0]).next()[0]).slideDown(400);
            }
    	}else{
    		 $($($(ele).parent()[0]).next()[0]).slideUp(400);
    		 $(ele).html("<%=rb.getString("ChaKan")%>");
    	}
    }
  	function toSoftwareFile(versionId,upgradeType){
  		 if(upgradeType == "enodeb"){
    		 isJumpToPage = {code:'enodeb',vid:versionId}
    		 goMenuPage("1004");
    	  } 
  		 if(upgradeType == "enodeb_patch"){
  			isJumpToPage = {code:'enodeb_patch',vid:versionId}
   		 	goMenuPage("1004");
  		 }
  		 if(upgradeType == "enodeb_uboot"){
  			isJumpToPage = {code:'enodeb_uboot',vid:versionId}
   		 	goMenuPage("1004");
  		 }
  		if(upgradeType == "enodeb_fpga"){
  			isJumpToPage = {code:'enodeb_fpga',vid:versionId}
   		 	goMenuPage("1004");
  		 }
  		 if(upgradeType == "cpe_odu"){
  			isJumpToPage = {code:'cpe_odu',vid:versionId}
   		 	goMenuPage("7004");
  		 }
  		 if(upgradeType == "cpe_idu"){
  			isJumpToPage = {code:'cpe_idu',vid:versionId}
   		 	goMenuPage("7004");
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
    function goUpgrade(typeNumber,versionId){
    	$("#newFileDetails").slideUp(300,function(){
    		$("#upgrade_container").slideDown(400,function(){
    			$("#upgrade_container").panel({
    				href:"${ctx}/task/upgrade/toNewUpgrade.action",
    				queryParams:{
    					type:typeNumber,
    					versionId:versionId
    				}
    			})
    		})
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