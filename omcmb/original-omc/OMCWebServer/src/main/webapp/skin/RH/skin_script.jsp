<meta http-equiv="Content-Type" content="text/html; charset=utf-8"/>
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
	<form id="lockform" >  
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
	<form>
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

<div id="mainCover" class="window-mask" style="display:none;width:2000px;height:1800px;position:absolute;left:0px;top:0px;z-index:1010"></div>
<div id="modal" class="modal"></div>

<div id="winPasswdRulePrompt" class="easyui-window" title="<%=rb.getString("MiMaCeLueBiaoTi")%>"
		data-options="modal:true,split:true,closed:true,collapsible:false,minimizable:false,maximizable:false,
			width:400,height:150">
	<div class="easyui-layout" data-options="border:false,fit:true">
		<div region="center" data-options="border:false" style="text-align: center;padding: 15px;">
           	<span><%=rb.getString("MiMaChangDu")%><%=rb.getString("MaoHao")%></span>
           	<span id="passwdMinLengthPrompt" value=""/></span> -
           	<span id="passwdMaxLengthPrompt" value=""/></span>
           	<span><%=rb.getString("ZiFu")%></span>
           	<span><%=rb.getString("DouHao")%></span>
            
            <span><%=rb.getString("ZhangHuYouXiaoTianShu")%><%=rb.getString("MaoHao")%></span>
            <span id="passwdValidPeriodPrompt" value=""></span>
            <span><%=rb.getString("Tian")%></span>
            <span><%=rb.getString("JuHao")%></span>
		</div>
		<div region="south" data-options="border:false,height:47" style="text-align: center;padding: 10px">
			<a href="#" class="easyui-linkbutton" style="margin-right:15px;" onclick="sureModifyPass()"><%=rb.getString("QueDing")%></a>
			<a href="#" class="easyui-linkbutton" style="" onclick="cancelModifyPass()"><%=rb.getString("QuXiao")%></a>
		</div>
	</div>
</div>


<div id="winDefault" class="easyui-window" title=" "
	data-options="modal:true,split:true,closed:true,collapsible:false,minimizable:false,maximizable:false,width:900,height:500,resizable:false">
</div>

<div id="getCellInfoErrorTips" class="getCellInfo" style="display: none;">
	<span></span>
</div>
<div id="getCellInfoOffTips" class="getCellInfo" style="display: none;">
	<span></span>
</div>
<div id="getCellInfoOnTips" class="getCellInfo" style="display: none;">
	<span></span>
</div>
<div id="getCellInfoUpdateTips" class="getCellInfo" style="display: none;">
	<span></span>
</div>

<script type="text/javascript">
	$(document).click(function(e){
		parent.postMessage({msg:"mess"},"*")
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

    $(function () {
    	extendEasyui();

        combonFirstTimeAndLeftMenu();
		
        /*Ã¦ÂÂ¯Ã¥ÂÂ¦Ã¦ÂÂÃ¥Â¯ÂÃ§Â ÂÃ¥Â°ÂÃ¨Â¿ÂÃ¦ÂÂÃ¦ÂÂÃ©ÂÂ */
        //Ã¥ÂÂ¤Ã¦ÂÂ­Ã¦ÂÂ¯Ã¥ÂÂ¦Ã¦ÂÂ¯4AÃ§Â³Â»Ã§Â»Â
        if (isCloudCore == "false") {
	        $.post("${ctx}/system/sysuser/queryPaswdExpirePrompt.action", {}, function (data) {
	            if (data["Expire"].length > 0 && "${ispwdtitle}"!="true" && data["pwd_enable"] == "0") {
	                $.messager.alert(TiShi, data["Expire"]);
	                return;
	            }
	        }, "json");
        }
        
      //å¤æ­æ¯å¦æ¯å¼å®¹æµè§å¨
        var explorer = window.navigator.userAgent;
		var explorFlag = false;
		if(explorer.indexOf("Firefox") > 0){//ç«çæµè§å¨
			explorFlag = true;
		}else if(explorer.indexOf("Edg") > 0){//Edgeæµè§å¨
			explorFlag = true;
		}else if(explorer.indexOf("Mac") > 0){//Safariæµè§å¨
			explorFlag = true;
		}else if(isChrome()){//Googleæµè§å¨
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
		if(explorFlag){//æ­¤æµè§å¨æ¯æ¯æçæµè§å¨
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
   			 	eventAllBus.$emit("gomenupage","1004","","1004",false);
   			 }catch(e){}
    	  } 
  		 if(upgradeType == "enodeb_patch"){
  			isJumpToPage = {code:'enodeb_patch',vid:versionId,type:'view',file_type:file_type}
			 try{
 			 	eventAllBus.$emit("gomenupage","1004","","1004",false);
 			 }catch(e){}
  		 }
  		 if(upgradeType == "enodeb_uboot"){
  			isJumpToPage = {code:'enodeb_uboot',vid:versionId,type:'view',file_type:file_type}
			try{
				eventAllBus.$emit("gomenupage","1004","","1004",false);
			}catch(e){}
  		 }
  		if(upgradeType == "enodeb_fpga"){
  			isJumpToPage = {code:'enodeb_fpga',vid:versionId,type:'view',file_type:file_type}
			try{
				eventAllBus.$emit("gomenupage","1004","","1004",false);
			}catch(e){}
  		 }
  		 if(upgradeType == "cpe_odu"){
  			isJumpToPage = {code:'cpe_odu',vid:versionId,type:'view'}
			try{
				eventAllBus.$emit("gomenupage","7004","","7004",false);
			}catch(e){}
  		 }
  		 if(upgradeType == "cpe_idu"){
  			isJumpToPage = {code:'cpe_idu',vid:versionId,type:'view'}
			try{
				eventAllBus.$emit("gomenupage","7004","","7004",false);
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
	   			 	eventAllBus.$emit("gomenupage","1004","","1004",false);
	   			 }catch(e){}
	    	 }else{
	    		 isJumpToPage = {vid:versionId,file_type:file_type,type:'upgrade'}
	   			 try{
	   			 	eventAllBus.$emit("gomenupage","7004","","7004",false);
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
		}
	}
	window.addEventListener("message",ChangeLanguageByCloudcore);
	
</script>

</html>