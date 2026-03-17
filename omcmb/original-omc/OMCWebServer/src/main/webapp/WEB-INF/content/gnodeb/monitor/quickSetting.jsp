<%@ page import="java.util.Locale"%>
<%@ page contentType="text/html;charset=UTF-8" %>
<style>
	#quickSteting .el-card__footer {
		border:none;
		border-top:1px solid #E9E9E9;
	}
	.left-title-list li{
		height:36px;
		line-height:36px;
		padding:0 20px;
		cursor: pointer;
		border-bottom: 1px solid #EEE;
		word-break: keep-all;
	}
	.left-title-list li.active {
		color: #4D84FF;
		background-color: #EDF6FF;
	}
</style>

<!--基站快速设置浮层 -->
<div id="enbSetting_slide" class='slidebarPanels flex-ctn panelDefault' style="display: none;position:absolute;background:#FFFFFF;z-index:100;top: 0;bottom:0px;left: 0;right:0px;">
    <div class="tabsTitle">
        <span tabtit="quickSteting" onclick="turnTabs(this)" class="active"><%=rb.getString("SheZhi")%></span>		
    </div>
    <div class="tabsContentDiv">
        <div class="quickSteting" id="quickSteting" style="display: flex;">
            <div style="border-right: 1px solid #EEE;min-width:110px">
                <div class="slidebarSecondTabsTitle">
                    <ul id="quick_setting_nav" class="left-title-list titleTabsList">
                    </ul>
                </div>
            </div>
            <div id="setting_form_cnt" style="flex: auto;overflow: auto;display:flex;flex-direction: column;position:relative;">
                <div class="slidebarTitleDiv el-card__header" style="padding-left:0px;">
                    <ul class="slidebarTitleContainer">
                        <li class="default"> </li>
                    </ul>
                    <div class="el-icon-circle-refresh el-icon form_bt_refresh" style="position:absolute;right:70px;font-size:25px;"></div>
                    <div class="el-icon el-icon-circle-goback form_bt_reback" style="position:absolute;right:80px;font-size:25px;" onclick="openPropsPanel(false);"></div>
                    <div class="el-icon-circle-close el-icon" style="position:absolute;right:20px;font-size:25px;" onclick="closeSettingPanel(closeTrue)"></div>
                </div>
                <div class='el-card__body' style="flex: auto;overflow: auto;border: none;">
                    <form id="enbSetting_slide_body" class="slide-body form-ctn" style="flex:1 auto;overflow: auto;height:100%;margin:0px;width:auto;padding:0px;background:#fff;">
                    </form>
                </div>
            </div>
        </div>
    </div>
</div>

<script>
    // 基站设置国际化
	$.renderDefaultOptions.system = '${ctx}'?'${ctx}/':'';
	Render.submitTxt = '<%=rb.getString("QueDing")%>';
	Render.resetTxt = '<%=rb.getString("QuXiao")%>';
	Render.propOkTxt = '<%=rb.getString("QueDing")%>';
	Render.propCancelTxt = '<%=rb.getString("QuXiao")%>';
	var TISHI = '<%=rb.getString("TiShi")%>',
		CHENGGONG = '<%=rb.getString("ChengGong")%>';
	var enbPlatform = '';
		
	Render.status.success = CHENGGONG;
	Render.status.operation = '<%=rb.getString("CaoZuo")%>';
	Render.status.tips = '<%=rb.getString("QueRen")%>';
	Render.status.confirm = '<%=rb.getString("QueRenCheXiao")%>';
	Render.status.remove = '<%=rb.getString("QueRenShanChu")%>';
	var closeTrue = true
	var addEdit = false
	// 基站设置提交
	Render.submit = function(){
		var valid = Render.valid();
		if(!valid) return;
		
		var params = Render.getFormDatas($('#enbSetting_slide_body'));
		// 判定是否有修改项，无则提示且不提交
		if(isEmptyJson(params)) {
			showMsg('prompt_msg','<%=rb.getString("CanShuZhiMeiYouBianHua")%>');
			return;
		}
		// 判定是否有重启项
		if(Render.validResult.reboot){
			var tipContent = [
					"<%=rb.getString("JiZhanChongQiTiShi")%>",
					"<br/><br/>",
					"<input id='reboot_confirm_status' type='checkbox' />",
					" <label for='reboot_confirm_status' style='font-size: 14px;color: #1DA3FC;cursor: pointer;'><%=rb.getString("SheZhiHouChongQi")%></label>"
				].join("");
			var msger = $.messager.confirm("<%=rb.getString("QueRen")%>", tipContent, function (r) {
				if (r) {
					/* 重启勾选判断 */
					var needReboot = false,
						rebootCkbox = $('#reboot_confirm_status',msger);
					if(rebootCkbox.length && rebootCkbox.prop('checked')){
						needReboot = true;
					}
					msger = null;
					
					$('#setting_form_cnt').addClass('loading');
					var rowCode = select_row_data.small_cell_code;
					
					$.post('${ctx}/cell/quicksettings/saveParamValue.action?smallCellCode='+rowCode+'&isGnb=1',
						{params: JSON.stringify(params)},
						function(data){
							if(data.success){
								showMsg('success_msg',"<%=rb.getString("ChengGong")%>");
								$('#tableHomeCellList').datagrid('reload');
							
								// 勾选重启，下发重启指令
								if(needReboot) {
									$.post("${ctx}/cell/cpeinfos/cellReboot.action", {cell_code: rowCode}, function (data) {
										if (!data["success"]) {
											showMsg('error_msg',data["message"]);
										}
									}, "json");
								}
								addEdit = true;
								closeSettingPanel();
							}else{
								// 设置失败相关提示
								if(data.validMsg){
									$.messager.alert(TISHI,data.validMsg,'warning');
								}else{
									$.messager.confirm(TISHI,'<%=rb.getString("JiZhanSheZhiShiBai")%>',function(r){
										if(r) closeSettingPanel();
									});
								}
								addEdit = false;
							}
							Render.validResult.reboot = false;
							$('#setting_form_cnt').removeClass('loading')
						},'json');
				}
			}).addClass("seriousConfirm");
		}else{
			$('#setting_form_cnt').addClass('loading');
			var rowCode = select_row_data.small_cell_code;
			$.post('${ctx}/cell/quicksettings/saveParamValue.action?smallCellCode='+rowCode,
				{params: JSON.stringify(params)},
				function(data){
					if(data.success){
						// $('.form-operations .success').addClass('show');
						showMsg("success_msg",'<%=rb.getString("ChengGong")%>')
						$('#tableHomeCellList').datagrid('reload');
						closeSettingPanel();
						if(Render.validResult.reboot) {
							$.post("${ctx}/cell/cpeinfos/cellReboot.action", {cell_code: rowCode}, function (data) {
								if (!data["success"]) {
									showMsg('error_msg',data["message"]);
								}
							}, "json");
						}
						 addEdit = true
					}else{
						if(data.validMsg){
							$.messager.alert(TISHI,data.validMsg,'warning');
						}else{
							$.messager.confirm(TISHI,'<%=rb.getString("JiZhanSheZhiShiBai")%>',function(r){
								if(r) closeSettingPanel();;
							});
						}
						 addEdit = false
					}
					
					Render.validResult.reboot = false;
					$('#setting_form_cnt').removeClass('loading')
				},'json');
		}
	}
	// 基站设置页取消
	Render.reset = function(){
		var params = Render.getFormDatas($('#enbSetting_slide_body')),toClose=false;
		var edit = !isEmptyJson(params)
		if(edit){
			if(!addEdit){
				closeTrue =true
			}else{
				closeTrue =false
			}
		}else{
			closeTrue = true
		}
		closeSettingPanel(closeTrue);
	}
	// 初始化快速设置同步事件
	$('#enbSetting_slide .form_bt_refresh').off('click').on('click',function(){
		var slider = $('#enbSetting_slide'),
			sliderForm = $('#setting_form_cnt'),
			postData = slider.data('params');
		// 设置等待蒙层
		sliderForm.addClass('loading');
		// 发起同步指令
		$.ajax({
			url: '${ctx}/cell/quicksettings/sync.action',
			data: postData,
			type: 'post',
			dataType: 'json',
			success: function(data){
				// 取消等待蒙层
				sliderForm.removeClass('loading');
			},
			error: function(data){
				sliderForm.removeClass('loading');
			}
		});
	});
	// 基站设置输入域的校验提示国际化
	Render.messages = function(opts){
		var value = opts.value || '';

		if(opts.type=='select'){
			var sDom = $('#'+opts.name,$('.form-ctn'));
			if(opts.typeFlag == 'ipsec'){
				sDom = $('[comboname="'+opts.name+'"]',$('.form-ctn'));
			}
			var datas = sDom.combobox('getData');
			if(datas){
				datas.map(function(row){
					if(row.value==opts.value) value = row.text;
				});
			}
		}

		var msges = {
				valid: '',
				required: '<%=rb.getString("BiTian")%>',
				range:'<%=rb.getString("FanWei")%><%=rb.getString("MaoHao")%>' + opts.min + '-' + opts.max + '<%=rb.getString("JuHao")%><%=rb.getString("ZhengShu")%>',
				length: '<%=rb.getString("ChangDuJiaoYan")%>' + opts.minlength + '-' + opts.maxlength,
				reboot: '<%=rb.getString("ChongQiJiaoYan")%>' + value
			}
		
		return msges;
	}
	// 基站设置输入域blur事件触发的校验方法
	var eNbSetting = {
		/**
		* 开关切换
		* @param newVal{string}: 当前值
		* @param oldVal{string}: 历史值
		* @param props{object}: 属性集合
		**/
		switchChange: function(newVal,oldVal,props){
			eNbSetting.resetIp(newVal,oldVal,props);
			/* 级联联动关系 */
			if(props.cascade) {
				var cascade = eval('('+props.cascade+')');
				cascade.map(function(item){
					// 级联显示项
					if(item.show && newVal == item.value){
						item.show.map(function(id){
							var ctn = $('#'+id),
							pros = ctn.data('props');
							if(pros.type=='select'){
								var origVal = ctn.combobox('getValue');
								ctn.combobox('setValue','').combobox('setValue',origVal)
							}
						});
					}
					// 级联隐藏项
					if(item.hide && newVal == item.value){
						item.hide.map(function(id){
							setTimeout(function(){
								$('#'+id).parents('.form-item').addClass('form-hidden');
							},10);
						});
					}
				})
			}
		},
		/**
		* 模式切换
		* @param newVal{string}: 当前值
		* @param oldVal{string}: 历史值
		* @param props{object}: 属性集合
		**/
		modleChange: function(newVal,oldVal,props){
			eNbSetting.resetIp(newVal,oldVal,props);
			/* 级联关系 */
			if(props.cascade) {
				var cascade = eval('('+props.cascade+')');
				cascade.map(function(item){
					if(item.show && newVal == item.value){
						item.show.map(function(id){
							var ctn = $('#'+id),
								pros = ctn.data('props');

							if(pros.type=='select'){
								var origVal = ctn.combobox('getValue');
								ctn.combobox('setValue','').combobox('setValue',origVal)
							}
						});
					}
				})
			}
		},
		/**
		* 表格数据加载前事件
		* @param param{object}: 查询参数
		**/
		onBeforeLoad: function(param){
			var rowCode = select_row_data.small_cell_code;
			param['smallCellCode'] = rowCode;
		},
		/**
		* ipsec添加事件
		* @param opts{object}: 属性集合
		**/
		ipsecAdd: function(opts){
			var tb = $('#F3965F1B73440F98C800591D56848CD9');
			if($('#66C118DE22B6A1146D3A1A5FF7B66BC3').length) tb = $('#66C118DE22B6A1146D3A1A5FF7B66BC3');
			if($('#EF12ED11C7A3167C15CCF78EF40ABF5C').length) tb = $('#EF12ED11C7A3167C15CCF78EF40ABF5C');
			if($('#B20E15F6543059F57AE7476FE0679C81').length) tb = $('#B20E15F6543059F57AE7476FE0679C81');
			if($('#0D3FD79C1A7FACBDA596A3F7E2B269DA').length) tb = $('#0D3FD79C1A7FACBDA596A3F7E2B269DA');
			
			tb.datagrid('unselectAll');
			// 添加的数据不超过2条
			var rows = tb.datagrid('getRows');
			if(rows.length>=2) {
				return ;
			}
			
			var form = $('<form id="ipsecForm" class="flex-ctn" operateType="add" style="height: 100%;"></form>');
			$('.form-ctn .form-props').html('').append(form);
			// 加载ipsec的jsp片段
			form.load($.renderDefaultOptions.system+opts.propsUrl,function(data){
				$.parser.parse(this);
				openPropsPanel();
				eNbSetting.changeTitle(opts.label);
			});
		},
		/**
		* ipsec修改事件
		* @param opts{object}: 属性集合
		* @param row{object}: 表格行数据
		**/
		ipsecEdit: function(opts,row){
			var form = $('<form id="ipsecForm" class="flex-ctn" operateType="edit" style="height: 100%;"></form>');
			$('.form-ctn .form-props').html('').append(form);
			// 加载ipsec的jsp片段
			form.load($.renderDefaultOptions.system+opts.propsUrl,function(data){
				$.parser.parse(this);
				openPropsPanel();
				eNbSetting.changeTitle(opts.label);
			});
		},
		/**
		* 标题切换
		* @param title{string}: 标题文本
		**/
		changeTitle: function(title){
			var tdom = $('#enbSetting_slide .slidebarTitleContainer .default');
			tText = tdom.html();

			if(tdom.data('old')){
				
			}else{
				tdom.data('old',tText);
			}

			if(title == false) tdom.html(tdom.data('old'));
			else tdom.html(title);
		},
		/**
		* ipsec表格数据处理
		* @param data{object}: 表格数据
		**/
		ipsecLoadFilter_qc: function(data){
			data.rows = data.rows.map(function(row){
				row._remove = false;
				return row;
			});

			return data;
		},
		/**
		* mme ip添加校验
		* @param props{object}: 属性集合
		**/
		mmeIpClick: function(props){
			var value = $('#'+props.name+'_show').textbox('getValue');

			var ctner = $('#'+props.name+'_show').parents('.form-item');
			// ip校验通过设值
			if(isValidIP(value)){
				var origVal = $('#'+props.name).textbox('getValue');
				if(origVal){
					var list = origVal.split(','),
						existed = false;
					
					if(list.includes(value.trim())) {
						existed = true;
					}
					
					if(existed) return;
					else origVal += ','+value;
				}else{
					origVal = value;
				}

				$('#'+props.name).textbox('setValue',origVal);
				$('#'+props.name+'_show').textbox('setValue','');
				$('#'+props.name).next().find('input').blur();
			}
			if(value == ''){
				ctner.attr('data-msg','<%=rb.getString("IPShuRuTiShi")%>').addClass('invalid');
				setTimeout(function(){
					$('#'+props.name).next().find('input').blur();
				},3000)
			}
		},
		/**
		* 校验ip
		* @param e{event}: 鼠标事件
		* @param props{object}: 属性集合
		**/
		ipValid: function(e,props){
			var value = $('#'+props.name).textbox('getValue'),
				nowVal = $('#'+props.name.replace('_show','')).textbox('getValue'),
				originVal = $('#'+props.name.replace('_show','')).textbox('options').originalValue;
			
			var ctner = $('#'+props.name).parents('.form-item');
			// ip 是否重复
			if(isValidIP(value)){
				var list = nowVal.split(','),
					existed = false;
				
				list.map(function(ipItem){
					if(ipItem.indexOf(value.trim())>=0) existed = true;
				});

				if(existed && props.button) return '<%=rb.getString("YiCunZai")%>';
				else {
					if(props.button) $('#'+props.name.replace('_show','')).next().find('input').blur();
					return '';
				}
			}else if(value != '' && value!=originVal){
				return '<%=rb.getString("IPDiZhiFeiFa")%>';
			}else{
				if(props.button) $('#'+props.name.replace('_show','')).next().find('input').blur();
			}
		},
		/**
		* ip值改变事件
		* @param val{string}: 当前值
		* @param oldVal{string}: 历史值
		* @param props{object}: 属性集合
		**/
		ipChange: function(val,oldVal,props){
			var ctner = $('#'+props.name).parents('.form-item');
			$('.flex-ctn-row',ctner).remove();

			if(val){
				var arr = val.split(',');
				arr.map(function(item){
					if(item) eNbSetting.addSuffix(item,ctner);
				});
			}
		},
		/**
		* 添加后缀
		* @param val{string}: 当前值
		* @param ctner{dom}: dom容器节点
		**/
		addSuffix: function(val,ctner){
			var sufCtn = $('.flex-ctn-row',ctner);
			if(sufCtn.length==0){
				sufCtn = $('<div class="flex-ctn-row" style="padding-left: 3px;"></div>');
				ctner.append(sufCtn);
			}

			var suffix = $('<div class="form-suffix"><span class="text">'+val+'</span> <span class="form-bt-remove el-icon el-icon-operation-delete"></span></div>');
			// 删除绑定事件
			suffix.find('.form-bt-remove').on('click',function(){
				suffix.remove();
				var tb = ctner.find('input[textboxname]:first'), ipval = '';
				
				$('.flex-ctn-row>.form-suffix',ctner).each(function(n,item){
					if(ipval) ipval += ',';
					ipval += $(item).find('.text').text();
				});
				tb.textbox('setValue',ipval);
				tb.next().find('input').blur();
			});

			sufCtn.append(suffix);
		},
		/**
		* pci校验
		* @param e{event}: 鼠标事件
		* @param props{object}: 属性集合
		**/
		pciBlur: function(e,props){
			var val = $('#'+props.name).textbox('getValue')||'',
				reg = /^\d(,\d)*$/,
				regRange = /^(\d+)..(\d+)$/,
				valid = true,
				regInt = /^-?\d+$/;
			
			if(val.indexOf(',')>=0){
				// 逗号分隔多值
				var arr = val.split(',');
				arr.map(function(value){
					if(isNaN(value) || value<0 || value>503) valid = false;
				});
			}else if(regRange.test(val)){
				// ..分隔多值
				var m = val.match(regRange),
					first = m[1],second=m[2];
				if(first<second){
					if(first<0 || first>503) valid = false;
					if(second<0 || second>503) valid = false;
				}else valid = false;
			}else{
				// 单值
				if(isNaN(val) || val<0 || val>503 || !regInt.test(val)) valid = false;
			}
			
			if(valid) return '';
			else return Render.messages(props)['range'];
		},
		/**
		* earfch转化
		* @param e{event}: 鼠标事件
		* @param props{object}: 属性集合
		**/
		earfchBlur: function(e,props){
			var titleDom = $('#'+props.name).parent().prev();
			titleDom.text(titleDom.text().replace(/\(\w+\)/g,''));

			var val = $('#'+props.name).numberbox('getValue'),
				sufVal = val?translateToFre(val):'',
				suffstr = sufVal?sufVal.substring(sufVal.indexOf('(')):'';
			
			$('#'+props.name).numberbox({suffix: suffstr.replace(')',')')});

			if(props.cascade) {
				try{
					var cascade = eval('('+props.cascade+')'),
						item = cascade[0];
					
					var relys = item.rely.split(','),
						bandVal = $('#'+relys[0]).textbox('getValue'),
						result = checkEarfcnByBand(val, bandVal);
					
					if(result.valid) {
						return '';
					}else {
						return '<%=rb.getString("FanWei")%>: ' + JSON.stringify(result.range);
					}
				}catch(e){}
			}
		},
		/**
		* earfch聚焦事件
		* @param e{event}: 鼠标事件
		**/
		earfchFocus: function(e){
			var tb = $(e.target).parent().prev();
			var val = tb.numberbox('getValue');
			tb.numberbox('setText',val);
		},
		/**
		* plmn校验
		* @param e{event}: 鼠标事件
		* @param props{object}: 属性集合
		**/
		plmnBlur: function(e,props){
			var val = $('#'+props.name).textbox('getValue')||'',
				inValid = false,
				regInt = /^-?\d+$/;
			
			if(val && !regInt.test(val)) inValid = true;
			if(val){
				var pureVal = val.replace('-','');
				if(pureVal.length>6) inValid = true;
				if(pureVal.length<5) inValid = true;
			}
			
			if(inValid) return '<%=rb.getString("PLMNFanWeiTiShi")%>';
		},
		/**
		* imsi ip变动更新
		* @param newVal{string}: 当前值
		* @param oldVal{string}: 历史值
		* @param props{object}: 属性集合
		**/
		imsiIpChange: function(newVal,oldVal,props){
			var ctn = $('#'+props.name).parents('.form-item');
			$('.flex-ctn-row',ctn).remove();

			if(newVal){
				newVal.split(',').map(function(ipVal){
					eNbSetting.addImsiIp(ipVal,ctn);
				});
			}
		},
		/**
		* imsi ip校验
		* @param e{event}: 鼠标事件
		* @param props{object}: 属性集合
		**/
		imsiIpBlur: function (e,props){
			var leftVal = $('#'+props.name+'_left').textbox('getValue'),
				rightVal = $('#'+props.name).textbox('getValue'),
				nowVal = $('#'+props.name.replace('_show','')).textbox('getValue');
			// 校验IMSI合法性
			if(leftVal && leftVal.length!=15 || isNaN(leftVal)) return '<%=rb.getString("LGWImsiChangDuCuoWu")%>';
			// 校验ip合法性
			if(rightVal && !isValidIP(rightVal)){
				return '<%=rb.getString("IPDiZhiFeiFa")%>';
			}
			// imsi ip输入合法时
			if(leftVal && rightVal && leftVal.length==15 && isValidIP(rightVal)){
				var list = nowVal.split(','),
					combVal = leftVal +'+'+ rightVal
					existed = false;
				list.map(function(ipItem){
					if(ipItem.indexOf(combVal.trim())>=0) existed = true;
				});
				if(existed && props.button) return '<%=rb.getString("YiCunZai")%>';
			}
		},
		/**
		* spset校验
		* @param e{event}: 鼠标事件
		* @param props{object}: 属性集合
		**/
		ipsetBlur: function(e,props){
			var val = $('#'+props.name).textbox('getValue');

			if(val && !isValidIP(val)){
				return '<%=rb.getString("IPDiZhiFeiFa")%>';
			}
		},
		/**
		* ip范围左域校验
		* @param e{event}: 鼠标事件
		* @param props{object}: 属性集合
		**/
		leftIpBlur: function(e,props){
			var options = $('#'+props.name).textbox('options'),
				ipSelf = $('#'+props.name).textbox('getValue'),
				ipRight = '',
				origVal = options.originalValue;

			var msg = eNbSetting.staticIpBlur(e,props,ipSelf);
			if(msg) return msg;
			/* 与右ip比较 */
			if(props.cascade) {
				var cascade = eval('('+props.cascade+')');
				cascade.map(function(item){
					var relys = item.rely.split(',');
					ipRight = $('#'+relys[2]).textbox('getValue');
				})
			}
			if(eNbSetting.compareIp(ipSelf,ipRight)) return '<%=rb.getString("IPYingXiaoYu")%>';
		},
		/**
		* ip范围右域校验
		* @param e{event}: 鼠标事件
		* @param props{object}: 属性集合
		**/
		rightIpBlur: function(e,props){
			var options = $('#'+props.name).textbox('options'),
			ipSelf = $('#'+props.name).textbox('getValue'),
			ipLeft = '',
			origVal = options.originalValue;

			var msg = eNbSetting.staticIpBlur(e,props,ipSelf);
			if(msg) return msg;
			/* 与左ip比较 */
			if(props.cascade) {
				var cascade = eval('('+props.cascade+')');
				cascade.map(function(item){
				var relys = item.rely.split(',');
				ipLeft = $('#'+relys[2]).textbox('getValue');
				})
			}
			if(eNbSetting.compareIp(ipLeft,ipSelf)) return '<%=rb.getString("IPYingDaYu")%>';
		},
		/**
		* 静态ip校验
		* @param e{event}: 鼠标事件
		* @param props{object}: 属性集合
		* @param value{string}: 当前值
		**/
		staticIpBlur: function(e,props,value){/* 左右ip的依赖的 ip和子网掩码校验 */
			var invalid = false,
				msg = '';
			if(props.cascade) {
				var cascade = eval('('+props.cascade+')');
				cascade.map(function(item){
					var relys = item.rely.split(',');
					var ip = $('#'+relys[0]).textbox('getValue'),
						mask = $('#'+relys[1]).combobox('getValue'),
						minIp = getLowAddr(ip,mask),
						maxIp = getHighAddr(ip,mask);
					// 合法校验
					if(ip && mask){
						if(isValidIP(ip)){
							if(eNbSetting.compareIp(value,minIp) && eNbSetting.compareIp(maxIp,value)){

							}else{
								invalid = true;
								msg = '<%=rb.getString("IPFanWei")%> '+minIp+'-'+maxIp;
							}
						}else{
							invalid = true;
							msg = '<%=rb.getString("IPDiZhiFeiFa")%>';
						}
					}else{
						invalid = true;
						msg = '<%=rb.getString("IPMaskBiTian")%>';
					}
				})
			}
			if(invalid) return msg;
		},
		/**
		* ip大小比对
		* @param ipA{string}: ip值
		* @param ipB{string}: ip值
		**/
		compareIp: function(ipA, ipB){
			ipA = ipA.split('.').map(function(item){
				return padLeft(item,3,'0');
			});
			ipB = ipB.split('.').map(function(item){
				return padLeft(item,3,'0');
			});
			return ipA.join('')>=ipB.join('');

			function padLeft (str, len, charStr) {
				var s = str + '';
				return new Array(len - s.length + 1).join(charStr,  '') + s;
			}
		},
		/**
		* imsi ip添加逻辑及校验
		* @param props{object}: 属性集合
		**/
		imsiIpClick: function (props){
			var ctn = $('#'+props.name).parents('.form-item'),
				hideVal = $('#'+props.name).textbox('getValue'),
				leftVal = $.trim($('#'+props.name+'_show_left').textbox('getValue')),
				rightVal = $.trim($('#'+props.name+'_show').textbox('getValue'));

			if(leftVal && !isNaN(leftVal) && leftVal.length==15 && rightVal && isValidIP(rightVal)) {
				var isValid = true;
				if(props.cascade) {
					// 依赖项校验
					var cascade = eval('('+props.cascade+')');
					cascade.map(function(item){
						var relys = item.rely.split(','),
							leftIp = $.trim($('#'+relys[0]).textbox('getValue')),
							rightIp = $.trim($('#'+relys[1]).textbox('getValue'));
						if(isValidIP(leftIp) && isValidIP(rightIp) && eNbSetting.compareIp(rightVal,leftIp) && eNbSetting.compareIp(rightIp,rightVal)){
						}else {
							isValid = false;
							ctn.attr('data-msg','<%=rb.getString("IPFanWei")%> '+leftIp+'-'+rightIp).addClass('invalid');
							setTimeout(function(){
								$('#'+props.name).next().find('input').blur();
							},3000);
						}
					})
				}
				if(isValid){
					var combVal = leftVal +'+'+rightVal;
					if(hideVal){
						var list = hideVal.split(','),
							existed = false;
						list.map(function(ipItem){
							if(ipItem.indexOf(combVal.trim())>=0) existed = true;
						});
						if(existed) return;
						else hideVal += ','+combVal;
					}else{
						hideVal = combVal;
					}

					eNbSetting.addImsiIp(combVal,ctn);
					$('#'+props.name).textbox('setValue',hideVal);
					$('#'+props.name+'_show_left').textbox('setValue','');
					$('#'+props.name+'_show').textbox('setValue','');
				}
			}
			if(!leftVal || !rightVal){
				ctn.attr('data-msg','<%=rb.getString("IPAndIMSITiShi")%>').addClass('invalid');
				setTimeout(function(){
					$('#'+props.name).next().find('input').blur();
				},3000);
			}
		},
		/**
		* imsi ip添加
		* @param val{string}: 当前值
		* @param ctn{dom}: dom容器节点
		**/
		addImsiIp: function (val,ctn){
			var suffCtn = $('.flex-ctn-row',ctn);
			if(suffCtn.length==0){
				suffCtn = $('<div class="flex-ctn-row" style="padding-left: 3px;"></div>');
				ctn.append(suffCtn);
			}

			var suffix = $('<div class="form-imsi-suffix"><span class="text">'+val+'</span><span class="form-bt-remove el-icon el-icon-operation-delete"></span></div>');
			// 删除按钮事件绑定
			suffix.find('.form-bt-remove').on('click',function(){
				suffix.remove();
				var tb = ctn.find('input[textboxname]:first'), ipval = '';

				$('.flex-ctn-row>.form-imsi-suffix',ctn).each(function(n,item){
					if(ipval) ipval += ',';
					ipval += $(item).find('.text').text();
				});
				tb.textbox('setValue',ipval);
				tb.next().find('input').blur();
			});

			suffCtn.append(suffix);
		},
		/**
		* ip重置
		* @param newVal{string}: 当前值
		* @param oldVal{string}: 历史值
		* @param props{object}: 属性集合
		**/
		resetIp: function(newVal,oldVal,props){
			if(props.cascade) {
				var cascade = eval('('+props.cascade+')');
				cascade.map(function(item){
					/* 重置级联项并触发校验 */
					if(item.target){
						item.target.split(',').map(function(tarId){
							var ctn = $('#'+tarId),
							pros = ctn.data('props'),
							fnName = $.domRenderDefaults[pros.type],
							orvalue = ctn[fnName]('options').originalValue;
							
							if(pros.type == 'range') {
								var cas = eval('('+pros.cascade+')'),
									relys = cas[0].rely.split(','),
									leftIP = $('#'+relys[0]).textbox('getValue'),
									rightIP = $('#'+relys[1]).textbox('getValue');
								// 判断依赖ip范围是否合法，并过滤不在此范围的IMSI+IP
								if(isValidIP(leftIP) && isValidIP(rightIP)) {
									var rangeValue = ctn.textbox('getValue');
									if(rangeValue) {
										var result = rangeValue.split(',').filter(function(item){
												var imsiIP = item.split('+')[1];
												return eNbSetting.compareIp(imsiIP,leftIP) && eNbSetting.compareIp(rightIP,imsiIP);
											});
										ctn[fnName]('setValue',result.join(','));
									}
								}else {
									ctn[fnName]('setValue',orvalue);
								}
							}

							//ctn[fnName]('setValue',orvalue);
							ctn.next().find('input').blur();
						});
					}
					/* 触发依赖项校验 */
					if(item.rely){
						item.rely.split(',').map(function(relyId){
							$('#'+relyId).next().find('input').blur();
						});
					}
				})
			}
		},
		/**
		* imsi ip重置
		* @param ctn{dom}: dom容器节点
		**/
		resetIMSIIP: function(ctn){
			var orvalue = $(ctn).textbox('options').originalValue;
			$(ctn).textbox('setValue',orvalue);
		},
		/**
		* LBT Time校验
		* @param e{event}: 鼠标事件
		* @param props{object}: 属性集合
		**/
		LBTTimeBlur:function(e,props){
			var reg = /^((20|21|22|23|[0-1]\d):[0-5]\d:[0-5]\d)$/;
			var value = $('#'+props.name).textbox('getValue');
			var nowVal = $('#'+props.name.replace('_show','')).textbox('getValue')
			if(value == ""){
				
			}else{
				if(reg.test(value)){
					
				}else{
					return "<%=rb.getString("LBTChuFaShiJianFanWei")%>";
				} 
			}
		},
		/**
		* 子网掩码校验
		* @param e{event}: 鼠标事件
		* @param props{object}: 属性集合
		**/
		maskValid: function(e,props){
			var regstr = /^((128|192)|2(24|4[08]|5[245]))(\.(0|(128|192)|2((24)|(4[08])|(5[245])))){3}$/,
				reg = new RegExp(regstr),
				value = $('#'+props.name).textbox('getValue'),
				originVal = $('#'+props.name.replace('_show','')).textbox('options').originalValue;
			
			var ctner = $('#'+props.name).parents('.form-item');
			if(reg.test(value)){// ip 是否重复
				if(props.button) $('#'+props.name.replace('_show','')).next().find('input').blur();
				return '';
			}else if(value != '' && value!=originVal){
				return '<%=rb.getString("QingShuRuHeFaDeYanMa")%>';
			}else{
				if(props.button) $('#'+props.name.replace('_show','')).next().find('input').blur();
			}
		}
	};

    function validMult(str){
		var valid = true, min = 0, max = 503,
			reg = /^(\d+)\.\.(\d+)$/, intReg = /^-?\d+$/;
		var arr = str.split(',');
		arr.map(function(item){
			item = item.trim();
			if(isNaN(item)){
				if(reg.test(item)){
					var m = item.match(reg), first = m[1], second = m[2];
					if(first-second<0){
						if(first<min || first>max || second<min || second>max) valid = false;
					}else valid = false;
				}else valid = false;
			}else if(item<min || item>max || !intReg.test(item)) valid = false;
		});
		return valid;
	}
	function checkEarfcnByBand(earfcn, band) {
		var rels = {
				1: [2110, 2170],      2: [1930, 1990],   3: [1805, 1880],   4: [2110, 2155],      5: [869, 894], 
				6: [875, 885],        7: [2620,2690],    8: [925, 960],     9: [1844.9, 1879.9],  10: [2110, 2170],
				11: [1475.9, 1495.9], 12: [729, 746],    13: [746, 756],    14: [758, 768],       17: [734, 746],
				18: [860, 875],       19: [875, 890],    20: [791, 821],    21: [1495.9, 1510.9], 22: [3510, 3590],
				23: [2180, 2200],     24: [1525, 1559],  25: [1930, 1995],  26: [859, 894],       27: [852, 869],
				28: [758, 803],       29: [717, 728],    30: [2350, 2360],  31: [462.5, 467.5],   32: [1452, 1496],
				33: [1900, 1920],     34: [2010, 2025],  35: [1850, 1910],  36: [1930, 1990],     37: [1880, 1930],
				38: [2570, 2620],     39: [1880, 1920],  40: [2300, 2400],  41: [2496, 2696],     42: [3400, 3600],
				43: [3600, 3800],     44: [703, 803],    45: [1447, 1467],  46: [5150, 5925],     47: [5855, 5925],
				48: [3550, 3700],     49: [3550, 3700],  50: [1432, 1517],  51: [1427, 1432],     52: [3300, 3400],
				53: [2483.5, 2495],   65: [2110, 2200],  66: [2110, 2200],  67: [738, 758],       68: [753, 783],
				69: [2570, 2620],     70: [1995, 2020],  71: [617, 652],    72: [461, 466],       73: [460, 465],
				74: [1475, 1518],     75: [1432, 1517],  76: [1427, 1432],  85: [728, 746],       87: [420, 425],
				88: [422, 427]
			},
			key = (band||'').trim(),
			range = rels[key],
			result = {
				valid: false,
				range: []
			};
		
		if(range) {
			var min = range[0], max = range[1];
			
			if(earfcn >= min && earfcn <= max) {
				result.valid = true;
			}
			
			result.range = range;
		}else {
			result.valid = true;
		}
		
		return result;
	}
</script>